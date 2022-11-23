use aws_lambda_events::event::dynamodb::{Event, EventRecord};
use aws_types::region::Region;
use futures::{stream, StreamExt};
use lambda_runtime::{service_fn, Error, LambdaEvent};
use serde::{Serialize, Deserialize};
use serde_dynamo::aws_lambda_events_0_7::from_item;

struct Client {
    client: appsync::Client,
    region: Region,
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

#[derive(Debug, Clone, Deserialize)]
struct BroadcastResponse {
    #[allow(dead_code)]
    #[serde(rename="publishMatch")]
    publish_match: Option<PublishMatchResponse>,
}

#[derive(Debug, Clone, Deserialize)]
struct PublishMatchResponse {
    #[allow(dead_code)]
    #[serde(rename = "sessionId")]
    session_id: String,
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
    async fn new() -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let config = aws_config::load_from_env().await;
        let region = config.region().ok_or("No region in config")?.clone();
        let api = std::env::var("API_URL")?;
        let client = appsync::Client::new(config, region.clone(), api).await?;
        Ok(Self { client, region })
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

    async fn broadcast_match(&self, item: Item) -> Result<Item, Box<dyn std::error::Error + Send + Sync>> {
        let req = appsync::GraphqlRequest{
            variables: item.clone(),
            query: BROADCAST_QUERY.to_string(),
        };

        let resp = self.client.query::<Option<BroadcastResponse>, Item>(req).await?;
        match resp.errors.iter().len() == 0 {
            true => Ok(item),
            false => {
                println!("{:?}", resp);
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
    let handler_func_closure = move |event: LambdaEvent<Event>| async move {
        client_ref.run(event).await
    };

    lambda_runtime::run(service_fn(handler_func_closure)).await?;
    Ok(())
}