use serde::{Deserialize, Serialize};
use aws_sdk_secretsmanager as secretsmanager;

#[derive(Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Response {
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

pub struct Client {
    token: String,
}

impl Client {
    pub async fn new(config: &aws_config::SdkConfig) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let secret_client = secretsmanager::Client::new(config);

        let secret_id = std::env::var("SECRET_ARN")?;
        let secret = secret_client.get_secret_value().secret_id(secret_id).send().await?;
        let token = secret.secret_string().ok_or("failed to get secret string")?;

        Ok(Self { token: token.to_string() })
    }

    pub async fn lookup(&self, ip: String) -> Result<Response, Box<dyn std::error::Error + Send + Sync>> {
        // todo: add token
        let resp = reqwest::get(format!("https://ipwho.is/{}", ip))
            .await?
            .json::<Response>()
            .await?;

        Ok(resp)
    }
}