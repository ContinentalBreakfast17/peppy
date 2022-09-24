use lambda_runtime::{service_fn, Error, LambdaEvent};
use serde::{Deserialize, Serialize};
use aws_sdk_secretsmanager as secretsmanager;

#[derive(Deserialize)]
struct Request {
    ip: String,
}

#[derive(Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
struct Response {
    pub ip: String,
    pub success: bool,
    #[serde(rename = "type")]
    pub type_field: String,
    pub continent: String,
    #[serde(rename = "continent_code")]
    pub continent_code: String,
    pub country: String,
    #[serde(rename = "country_code")]
    pub country_code: String,
    pub region: String,
    #[serde(rename = "region_code")]
    pub region_code: String,
    pub city: String,
    pub latitude: f64,
    pub longitude: f64,
    #[serde(rename = "is_eu")]
    pub is_eu: bool,
    pub postal: String,
    #[serde(rename = "calling_code")]
    pub calling_code: String,
    pub capital: String,
    pub borders: String,
}

struct Client {
    token: String,
}

impl Client {
    async fn new() -> Result<Self, Error> {
        let config = aws_config::load_from_env().await;
        let secret_client = secretsmanager::Client::new(&config);

        let secret_id = std::env::var("SECRET_ARN").unwrap();
        let secret = secret_client.get_secret_value().secret_id(secret_id).send().await?;
        let token = secret.secret_string().unwrap();

        Ok(Self { token: token.to_string() })
    }

    // examples: https://github.com/awslabs/aws-lambda-rust-runtime/tree/main/examples
    // todo: better logging
    async fn run(&self, event: LambdaEvent<Request>) -> Result<Response, Error> {
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
        .with_max_level(tracing::Level::INFO)
        // disable printing the name of the module in every log line.
        .with_target(false)
        // disabling time is handy because CloudWatch will add the ingestion time.
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
