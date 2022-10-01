use lambda_runtime::{Error};
use queue_processor::{Client, init, handler};

struct Processor {
    client: Client,
}

impl Processor {
    async fn new() -> Result<Self, Error> {
        match Client::new().await {
            Ok(client) => Ok(Self { client }),
            Err(e) => Err(e)
        }
    }
}

impl queue_processor::Processor for Processor {
    fn client(&self) -> &Client {
        &self.client
    }
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    init().await;
    let processor = Processor::new().await.expect("Failed to init processor");
    handler(&processor).await
}