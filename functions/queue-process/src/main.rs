use std::time::{SystemTime, UNIX_EPOCH};
use once_cell::sync::OnceCell;

use aws_lambda_events::event::dynamodb::Event;
use lambda_runtime::{service_fn, Error, LambdaEvent};
use aws_types::region::Region;
// use serde_dynamo::{from_item, from_items, to_item};
use aws_sdk_dynamodb as dynamodb;

struct Client {
    region:       String,
    queue_table:  String,
    lock_table:   String,
    lock_regions: Vec<String>,
    queue_client: dynamodb::Client,
    lock_clients: Vec<dynamodb::Client>,
}

const MAX_FAILURES: i32 = 5;
static AWS_CONFIG: OnceCell<aws_config::SdkConfig> = OnceCell::new();
static LOCK_REGIONS: OnceCell<String> = OnceCell::new();

impl Client {
    async fn new() -> Result<Self, Error> {
        let config = AWS_CONFIG.get().expect("No AWS config");
        let region = config.region().expect("No region in config");
        let queue_client = dynamodb::Client::new(&config);

        let lock_regions: Vec<String> = LOCK_REGIONS.get().expect("No lock regions").split(",").map(str::to_string).collect();
        let mut lock_clients: Vec<dynamodb::Client> = Vec::new();
        for region in lock_regions.clone() {
            let regional_config = dynamodb::config::Builder::from(config)
                    .region(Region::new(region))
                    .build();
            lock_clients.push(dynamodb::Client::from_conf(regional_config));
        }

        let queue_table = std::env::var("QUEUE_TABLE").expect("QUEUE_TABLE not set");
        let lock_table = std::env::var("LOCK_TABLE").expect("LOCK_TABLE not set");

        Ok(Self { region: region.to_string(), queue_table, lock_table, lock_regions, queue_client, lock_clients })
    }

    async fn run(&self, _event: LambdaEvent<Event>) -> Result<(), Error> {
        let now = SystemTime::now();
        let epoch = now
            .duration_since(UNIX_EPOCH)
            .expect("Y2Q");

        // first, check if the region is reasonably healthy
        // if it is not, make no attempt to process the stream (assuming stream will be processed by other regions)
        let healthy = self.check_health(epoch).await.expect("Failed to check region health");
        if !healthy {
            println!("region not healthy, abandoning stream");
            return Ok(());
        }
        println!("region is healthy");

        // next, obtain lock

        // finally, process queue

        Ok(())
    }

    async fn check_health(&self, epoch: std::time::Duration) -> Result<bool, Error> {
        for (index, lock_client) in self.lock_clients.iter().enumerate() {
            let region = self.lock_regions.iter().nth(index).expect("invalid region index");
            let items_result = lock_client.query()
                .table_name(&self.lock_table)
                .select(dynamodb::model::Select::Count)
                .key_condition_expression("#process = :this AND #error_time > :x_minutes_ago")
                .expression_attribute_names("#process", "process")
                .expression_attribute_names("#error_time", "sk")
                .expression_attribute_values(":this", dynamodb::model::AttributeValue::S(format!("queue#{}#{}", self.queue_table, self.region)))
                .expression_attribute_values(":x_minutes_ago", dynamodb::model::AttributeValue::S((epoch.as_secs() - 300).to_string()))
                .send().await;

            let items = match items_result {
                Ok(items) => items,
                Err(e) => {
                    println!("Failed to query {} - {:?}", region, e);
                    dynamodb::output::QueryOutput::builder().count(std::i32::MIN).build()
                }
            };

            if items.count == std::i32::MIN {
                if index < self.lock_clients.len() - 1 {
                    println!("trying next region...");
                    continue
                }
                println!("out of regions to try");
                return Ok(false);
            } else if items.count > MAX_FAILURES {
                println!("{}", items.count);
                println!("region has exceeded failure count");
                return Ok(false);
            }

            println!("failure count: {}", items.count);
            break
        }
        Ok(true)
    }
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    tracing_subscriber::fmt()
        .with_max_level(tracing::Level::INFO)
        // disable printing the name of the module in every log line.
        .with_target(false)
        // disabling time is handy because CloudWatch will add the ingestion time.
        .without_time()
        .init();

    let config = aws_config::load_from_env().await;
    AWS_CONFIG.set(config).unwrap();
    let lock_regions = std::env::var("LOCK_REGIONS").expect("LOCK_REGIONS not set");
    LOCK_REGIONS.set(lock_regions).unwrap();

    let client = Client::new().await?;
    let client_ref = &client;

    // Define a closure here that makes use of the shared client.
    let handler_func_closure = move |event: LambdaEvent<Event>| async move {
        client_ref.run(event).await
    };

    lambda_runtime::run(service_fn(handler_func_closure)).await?;
    Ok(())
}