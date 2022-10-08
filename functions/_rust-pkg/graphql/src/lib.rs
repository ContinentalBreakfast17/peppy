use aws_sig_auth::signer::{OperationSigningConfig, RequestConfig, SigV4Signer};
use aws_smithy_http::body::SdkBody;
use aws_smithy_http::byte_stream::ByteStream;
use aws_types::credentials::ProvideCredentials;
use aws_types::region::{Region, SigningRegion};
use aws_types::SigningService;
use serde::{Serialize, Deserialize};
use serde_json::json;
use std::time::SystemTime;

pub struct Client {
    region:     Region,
    signer:     SigV4Signer,
    config:     aws_config::SdkConfig,
    api:        String,
    api_client: reqwest::Client,
}

#[derive(Debug, Clone, Serialize)]
pub struct GraphqlRequest<Vars: serde::Serialize> {
    pub query: String,
    pub variables: Vars,
}

#[derive(Debug, Clone, Deserialize)]
pub struct GraphqlResponse<Data: serde::de::DeserializeOwned> {
    #[allow(dead_code)]
    #[serde(deserialize_with = "Data::deserialize")]
    pub data: Data,
    #[allow(dead_code)]
    #[serde(default)]
    pub errors: Vec<GraphqlError>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct GraphqlError {
    #[allow(dead_code)]
    pub message: String,
}

impl Client {
    pub async fn new(config: aws_config::SdkConfig, region: Region, api: String) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let signer = aws_sig_auth::signer::SigV4Signer::new();
        let api_client = reqwest::Client::new();

        Ok(Self { region: region, signer, config, api, api_client })
    }

    pub async fn query<Data: serde::de::DeserializeOwned, Vars: serde::Serialize>(&self, gql_req: GraphqlRequest<Vars>) -> Result<GraphqlResponse<Data>, Box<dyn std::error::Error + Send + Sync>> {
        let body = json!(gql_req).to_string();
        let sdk_body = SdkBody::from(body);

        let mut request = http::Request::builder()
            .method("POST")
            .uri(self.api.clone())
            .body(sdk_body)?;

        self.sign_request(&mut request).await?;

        let reqw = self.convert_req(request)?;
        let response = self.api_client.execute(reqw)
            .await?
            .json::<GraphqlResponse<Data>>()
            .await?;

        Ok(response)
    }

    async fn sign_request(
        &self,
        mut request: &mut http::Request<SdkBody>,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let credentials_provider = match self.config.credentials_provider() {
            Some(p) => p,
            None => return Err("failed to get credentials provider")?
        };

        let request_config = RequestConfig {
            request_ts: SystemTime::now(),
            region: &SigningRegion::from(self.region.clone()),
            service: &SigningService::from_static("appsync"),
            payload_override: None,
        };

        self.signer.sign(
            &OperationSigningConfig::default_config(),
            &request_config,
            &credentials_provider.provide_credentials().await?,
            &mut request,
        )?;

        Ok(())
    }

    fn convert_req(&self, req: http::Request<SdkBody>) -> Result<reqwest::Request, Box<dyn std::error::Error + Send + Sync>> {
        let (head, body) = req.into_parts();
        let url = head.uri.to_string();
        let body = {
            let stream = ByteStream::new(body);
            reqwest::Body::wrap_stream(stream)
        };

        let reqw = self.api_client
            .request(head.method, url)
            .headers(head.headers)
            .version(head.version)
            .body(body)
            .build()?;
        Ok(reqw)
    }
}

// #[cfg(test)]
// mod tests {
//     use super::*;

//     #[test]
//     fn it_works() {
//         let result = add(2, 2);
//         assert_eq!(result, 4);
//     }
// }
