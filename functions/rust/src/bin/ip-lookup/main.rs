use aws_sdk_secretsmanager as secretsmanager;
use lambda_runtime::{service_fn, Error, LambdaEvent};
use serde::{Deserialize, Serialize};

#[derive(Deserialize)]
struct Request {
    ip: String,
}

#[derive(Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
struct Response {
    ip: String,
    success: bool,
    #[serde(rename = "type")]
    type_field: String,
    continent: String,
    #[serde(rename = "continent_code")]
    continent_code: String,
    country: String,
    #[serde(rename = "country_code")]
    country_code: String,
    region: String,
    #[serde(rename = "region_code")]
    region_code: String,
    city: String,
    latitude: f64,
    longitude: f64,
    #[serde(rename = "is_eu")]
    is_eu: bool,
    postal: String,
    #[serde(rename = "calling_code")]
    calling_code: String,
    capital: String,
    borders: String,
}

struct Client {
    token: String,
}

impl Client {
    async fn new() -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let config = aws_config::load_from_env().await;
        let secret_client = secretsmanager::Client::new(&config);

        let secret_id = std::env::var("SECRET_ARN")?;
        let secret = secret_client.get_secret_value().secret_id(secret_id).send().await?;
        let token = secret.secret_string().ok_or("failed to get secret string")?;

        Ok(Self { token: token.to_string() })
    }

    async fn run(&self, event: LambdaEvent<Request>) -> Result<Response, Box<dyn std::error::Error + Send + Sync>> {
        let ip = event.payload.ip;
        println!("ip: {}", ip);

        // todo: add token
        let resp = reqwest::get(format!("https://ipwho.is/{}", ip))
            .await?
            .json::<Response>()
            .await?;

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