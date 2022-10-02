use aws_lambda_events::event::dynamodb::{Event, EventRecord};
use aws_sig_auth::signer::{OperationSigningConfig, RequestConfig, SigV4Signer};
use aws_smithy_http::body::SdkBody;
use aws_smithy_http::byte_stream::ByteStream;
use aws_types::credentials::ProvideCredentials;
use aws_types::region::{Region, SigningRegion};
use aws_types::SigningService;
use futures::{stream, StreamExt};
use lambda_runtime::{service_fn, Error, LambdaEvent};
use serde::{Serialize, Deserialize};
use serde_json::json;
use serde_dynamo::aws_lambda_events_0_7::from_item;
use std::time::SystemTime;

struct Client {
    region:     Region,
    signer:     SigV4Signer,
    config:     aws_config::SdkConfig,
    api:        String,
    api_client: reqwest::Client,
}

#[derive(Serialize)]
struct Response {
    #[serde(rename = "batchItemFailures")]
    pub failures: Vec<ItemId>,
}

#[derive(Clone, Serialize)]
struct ItemId {
    #[serde(rename = "itemIdentifier")]
    pub id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Item {
    #[serde(skip_deserializing)]
    event_id: String,
    #[serde(skip_deserializing)]
    table_name: String,
    #[serde(rename = "match")]
    id: String,
    queue: String,
    #[serde(rename = "sessionId")]
    session_id: String,
    players: Vec<Player>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Player {
    #[serde(alias="user")]
    #[serde(rename = "userId")]
    user_id: String,
    ip: String,
    #[serde(skip_serializing)]
    region: String,
}

#[derive(Debug, Clone, Serialize)]
struct GraphqlRequest<Vars: serde::Serialize> {
    query: String,
    variables: Vars,
}

#[derive(Debug, Clone, Deserialize)]
struct GraphqlResponse {
    // #[allow(dead_code)]
    // #[serde(default)]
    // data: std::collections::HashMap<String, Value>,
    #[serde(default)]
    errors: Vec<GraphqlError>,
}

#[derive(Debug, Clone, Deserialize)]
struct GraphqlError {
    #[allow(dead_code)]
    message: String,
}

const CONCURRENT_REQUESTS: usize = 4;
const BROADCAST_QUERY: &str = "
mutation ($queue: String!, $sessionId: ID!, $players: [PlayerInput!]!){
    publishMatch(queue: $queue, sessionId: $sessionId, players: $players) {
        __typename
        ... on Match {
            queue
            sessionId
            playerIds
            players {
                userId
                ip
            }
        }
    }
}
";

impl Client {
    async fn new() -> Result<Self, Error> {
        let config = aws_config::load_from_env().await;
        let region = config.region().expect("No region in config");
        let signer = aws_sig_auth::signer::SigV4Signer::new();
        let api = std::env::var("API_URL").expect("API_URL not set");
        let api_client = reqwest::Client::new();

        Ok(Self { region: region.clone(), signer, config, api, api_client })
    }

    async fn run(&self, event: LambdaEvent<Event>) -> Result<Response, Error> {
        // find records where at least one player is in this region (ignore deletes)
        let records: Vec<&EventRecord> = event.payload.records.iter().filter(|record| {
            match vec!["INSERT".to_string(), "MODIFY".to_string()].contains(&record.event_name) {
                false => false,
                true => match from_item::<Item>(record.change.new_image.clone()) {
                    Err(e) => {
                        println!("invalid item: {:?}", e);
                        false
                    },
                    Ok(item) => item.players.iter().any(|player| player.region == self.region.to_string()) 
                }
            }
        }).collect();

        // fetch the corresponding items out of the record
        let items: Vec<Item> = records.iter().map(|record| {
            let mut item: Item = from_item(record.change.new_image.clone()).unwrap();
            item.event_id = record.event_id.clone();
            item
        }).collect();

        // set up parallel execution
        let broadcast_results = stream::iter(items).map(|item| {
            async move {
                self.broadcast_match(item).await
            }
        }).buffer_unordered(CONCURRENT_REQUESTS);

        // wait on results
        let successes = broadcast_results.filter_map(|result| async move {
            match result {
                Ok(item) => {
                    println!("published match: {}", item.id);
                    Some(item.event_id)
                },
                Err(e) => {
                    println!("failed to publish match: {:?}", e);
                    None
                }
            }
        }).collect::<Vec<_>>().await;

        // figure out which records failed by filtering out successes
        let failures: Vec<ItemId> = records.iter().filter_map(|record| {
            match successes.contains(&record.event_id) {
                false => Some(ItemId{id: record.event_id.clone()}),
                true => None
            }
        }).collect();

        Ok(Response{failures: failures})
    }

    async fn broadcast_match(&self, item: Item) -> Result<Item, Box<dyn std::error::Error>> {
        let body = json!(GraphqlRequest{
            variables: item.clone(),
            query: BROADCAST_QUERY.to_string(),
        }).to_string();
        let sdk_body = SdkBody::from(body);

        let mut request = http::Request::builder()
            .method("POST")
            .uri(self.api.clone())
            .body(sdk_body)?;

        match self.sign_request(&mut request).await {
            Ok(_) => (),
            Err(e) => {
                println!("sign request error: {:?}", e);
                Err("failed to sign request")?
            }
        };

        let reqw = self.convert_req(request)?;
        let res = self.api_client.execute(reqw)
            .await?
            .json::<GraphqlResponse>()
            .await?;

        match res.errors.iter().len() == 0 {
            true => Ok(item),
            false => {
                println!("{:?}", res);
                Err("graphql call failed")?
            }
        }
    }

    async fn sign_request(
        &self,
        mut request: &mut http::Request<SdkBody>,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let credentials_provider = match self.config.credentials_provider() {
            Some(p) => p,
            None => return Err("failed to get credentials provider")?
        };

        let now = SystemTime::now();

        let request_config = RequestConfig {
            request_ts: now,
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

    fn convert_req(&self, req: http::Request<SdkBody>) -> Result<reqwest::Request, Box<dyn std::error::Error>> {
        let (head, body) = req.into_parts();
        let url = head.uri.to_string();

        let body = {
            // `SdkBody` doesn't currently impl stream but we can wrap
            // it in a `ByteStream` and then we're good to go.
            let stream = ByteStream::new(body);
            // Requires `reqwest` crate feature "stream"
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
    let handler_func_closure = move |event: LambdaEvent<Event>| async move {
        client_ref.run(event).await
    };

    lambda_runtime::run(service_fn(handler_func_closure)).await?;
    Ok(())
}