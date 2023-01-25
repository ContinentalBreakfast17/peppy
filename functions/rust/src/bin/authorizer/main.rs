use aws_lambda_events::event::appsync::AppSyncLambdaAuthorizerRequest;
use aws_lambda_events::event::appsync::AppSyncLambdaAuthorizerResponse;
use aws_sdk_dynamodb as dynamodb;
use lambda_runtime::{service_fn, Error, LambdaEvent};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

struct Client {
    dynamo_client: dynamodb::Client,
    table:         String,
    index:         String,
}

#[derive(Clone, Deserialize, Serialize)]
struct User {
    user: String,
    name: String,
}

impl Client {
    async fn new() -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let config = aws_config::load_from_env().await;
        let table = std::env::var("TABLE")?;
        let index = std::env::var("INDEX")?;
        let dynamo_client = dynamodb::Client::new(&config);
        Ok(Self { dynamo_client, table, index })
    }

    async fn run(&self, event: LambdaEvent<AppSyncLambdaAuthorizerRequest>) -> Result<AppSyncLambdaAuthorizerResponse<User>,  Box<dyn std::error::Error + Send + Sync>> {
        match self.is_authorized(event).await {
            Ok(resp) => match resp {
                // todo: should probably log something here
                Some(user) => Ok(self.authorized_response(user)),
                None => Ok(self.unauthorized_response())
            }
            Err(e) => {
                println!("failed to check auth: {:?}", e);
                Err(e)?
            }
        }
    }

    async fn is_authorized(&self, event: LambdaEvent<AppSyncLambdaAuthorizerRequest>) -> Result<Option<User>,  Box<dyn std::error::Error + Send + Sync>> {
        let play_key = match event.payload.authorization_token {
            Some(token) => token,
            None => {
                println!("no auth token");
                return Ok(None)
            }
        };

        let query_result = self.dynamo_client.query()
            .table_name(&self.table)
            .index_name(&self.index)
            .select(dynamodb::model::Select::AllAttributes)
            .key_condition_expression("#playKey = :playKey")
            .expression_attribute_names("#playKey", "playKey")
            .expression_attribute_values(":playKey", aws_sdk_dynamodb::model::AttributeValue::S(play_key))
            .send().await;

        match query_result {
            Ok(query_output) => match query_output.items {
                Some(users) => match serde_dynamo::from_items::<Vec<HashMap<String, dynamodb::model::AttributeValue>>, User>(users) {
                    // todo: not correct return
                    Ok(parsed_users) => {
                        if parsed_users.len() == 0 {
                            // no matching users
                            println!("no matching users");
                            Ok(None)
                        } else if parsed_users.len() > 1 {
                            // more than 1 matching user
                            // todo: should probably log users
                            println!("Found more than 1 user for playKey");
                            Ok(Some(parsed_users[0].clone()))
                        } else {
                            // exactly one matching user (what we expect)
                            println!("found user");
                            Ok(Some(parsed_users[0].clone()))
                        }
                    },
                    Err(e) => {
                        println!("Could not parse dynamo response");
                        Err(e)?
                    }
                },
                None => Ok(None)
            },
            Err(e) => {
                println!("failed to query queue");
                Err(e)?
            }
        }
    }

    fn authorized_response(&self, user: User) -> AppSyncLambdaAuthorizerResponse<User> {
        AppSyncLambdaAuthorizerResponse{
            is_authorized: true,
            resolver_context: HashMap::from([("user".to_string(), user)]),
            ttl_override: None,
            // could use this to deny ranked
            denied_fields: None,
        }
    }

    fn unauthorized_response(&self) -> AppSyncLambdaAuthorizerResponse<User> {
        AppSyncLambdaAuthorizerResponse{
            is_authorized: false,
            resolver_context: HashMap::from([]),
            ttl_override: None,
            denied_fields: None,
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
    let handler_func_closure = move |event: LambdaEvent<AppSyncLambdaAuthorizerRequest>| async move {
        client_ref.run(event).await
    };

    lambda_runtime::run(service_fn(handler_func_closure)).await?;
    Ok(())
}