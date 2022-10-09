use aws_lambda_events::event::cloudwatch_events::CloudWatchEvent;
use aws_sdk_dynamodb as dynamodb;
use lambda_runtime::{service_fn, Error, LambdaEvent};
use serde::{Serialize, Deserialize};

struct Client {
    client:        graphql::Client,
    region:        String,
    table:         String,
    dynamo_client: dynamodb::Client,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct HealthcheckResponse {
    healthcheck: Option<Healthcheck>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Healthcheck {
    id: String,
}

const HEALTCHECK_QUERY: &str = "
subscription($id: ID!){
    healthcheck(id: $id) {
        id
    }
}
";

impl Client {
    async fn new() -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let config = aws_config::load_from_env().await;
        let region = config.region().ok_or("No region in config")?.clone();
        let api = std::env::var("API_URL")?;
        let table = std::env::var("TABLE")?;
        let client = graphql::Client::new(config.clone(), region.clone(), api).await?;
        let dynamo_client = dynamodb::Client::new(&config);
        Ok(Self { client, region: region.to_string(), table, dynamo_client })
    }

    async fn run(&self, event: LambdaEvent<CloudWatchEvent>) -> Result<(),  Box<dyn std::error::Error + Send + Sync>> {
        println!("event id: {:?}", event.payload.id);
        let id = ksuid::Ksuid::generate();

        let req = graphql::GraphqlRequest{
            query: HEALTCHECK_QUERY.to_string(),
            variables: Healthcheck{id: id.to_base62()}
        };

        self.client.subscribe(req, process_subscription, 2000, 10000).await?;

        let completion_ts = ksuid::Ksuid::generate().time().sec;
        self.dynamo_client.update_item()
            .table_name(&self.table)
            .key("region", dynamodb::model::AttributeValue::S(self.region.clone()))
            .key("id", dynamodb::model::AttributeValue::S(format!("healthcheck#{}", id.to_base62())))
            .update_expression("SET #timestamp = :now, #status = :complete")
            .condition_expression("attribute_exists(#status) AND #status = :responded")
            .expression_attribute_names("#timestamp", "timestamp_completed")
            .expression_attribute_names("#status", "status")
            .expression_attribute_values(":now", dynamodb::model::AttributeValue::N(completion_ts.to_string()))
            .expression_attribute_values(":complete", dynamodb::model::AttributeValue::S("complete".to_string()))
            .expression_attribute_values(":responded", dynamodb::model::AttributeValue::S("responded".to_string()))
            .send().await?;

        Ok(())
    }
}

async fn process_subscription(response: graphql::GraphqlResponse<HealthcheckResponse>) -> Result<Option<()>, Box<dyn std::error::Error + Send + Sync>> {
    println!("{:?}", response);
    match response.errors.iter().len() == 0 {
        // exit as soon as we get one success
        true => Ok(None),
        false => Err("graphql call failed")?
    }
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    tracing_subscriber::fmt()
        .with_max_level(tracing::Level::WARN)
        .with_target(false)
        .without_time()
        .init();

    let client = Client::new().await?;
    let client_ref = &client;

    // Define a closure here that makes use of the shared client.
    let handler_func_closure = move |event: LambdaEvent<CloudWatchEvent>| async move {
        client_ref.run(event).await
    };

    lambda_runtime::run(service_fn(handler_func_closure)).await?;
    Ok(())
}