use lambda_runtime::{service_fn, Error, LambdaEvent};
use serde::{Deserialize};

#[derive(Deserialize)]
struct Request {
    ip: String,
}

struct Client {
    client: ip_lookup::Client,
}

impl Client {
    async fn new() -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let config = aws_config::load_from_env().await;
        let client = ip_lookup::Client::new(&config).await?;
        Ok(Self { client })
    }

    async fn run(&self, event: LambdaEvent<Request>) -> Result<ip_lookup::Response, Box<dyn std::error::Error + Send + Sync>> {
        let ip = event.payload.ip;
        println!("ip: {}", ip);
        let resp = self.client.lookup(ip).await?;
        println!("success");
        Ok(resp)
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
    let handler_func_closure = move |event: LambdaEvent<Request>| async move {
        client_ref.run(event).await
    };

    lambda_runtime::run(service_fn(handler_func_closure)).await?;
    Ok(())
}
