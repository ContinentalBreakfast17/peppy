// todo: implement the following
// - this function sends a subscription request (https://stackoverflow.com/questions/73317229/how-to-set-origin-header-to-websocket-client-in-rust)
// - app sync runs ip lookup function, then places item in healthcheck table
// - stream reads healthcheck table (regional items only), then pushes mutation
// - this function sees the message + terminates
// - we should through a lambda -> dynamo action in there somewhere, and maybe a transaction? just in case those break specifically (maybe just updating the healthcheck item?)
// match processor should read the cloudwatch alarm for its region + refuse to process if it is in alarm (to prevent a broken region from acquiring the lock)

use aws_lambda_events::event::cloudwatch_events::CloudWatchEvent;
use lambda_runtime::{service_fn, Error, LambdaEvent};
use serde::{Serialize, Deserialize};

struct Client {
    client: graphql::Client,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Healthcheck {
    id: String,
}

const HEALTCHECK_QUERY: &str = "
subscription ($id: ID!){
    healthcheck(id: $id) {
        id
    }
}
";

impl Client {
    async fn new() -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let config = aws_config::load_from_env().await;
        let api = std::env::var("API_URL").unwrap_or("API_URL not set");
        Ok(Self { client: graphql::Client::new(&config, api) })
    }

    async fn run(&self, event: LambdaEvent<CloudWatchEvent>) -> Result<(), Error> {
        println!("{:?}", event);
        let id = ksuid::Ksuid::generate();

        self.healthcheck(id).await
    }

    async fn healthcheck(&self, id: ksuid::Ksuid) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let req = GraphqlRequest{
            query: HEALTCHECK_QUERY.to_string(),
            vars: Healthcheck{id: id.to_base62()}
        };

        let resp = self.client.query::<Healthcheck>(req).await?;
        match res.errors.iter().len() == 0 {
            true => Ok(item),
            false => {
                println!("{:?}", res);
                Err("graphql call failed")?
            }
        }
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