use async_trait::async_trait;
use aws_lambda_events::event::dynamodb::Event;
use aws_sdk_dynamodb as dynamodb;
use aws_sdk_cloudwatch as cloudwatch;
use futures::{stream, StreamExt};
use lambda_runtime::{service_fn, Error, LambdaEvent};
use svix_ksuid::*;

pub struct Client<QueueItem: serde::de::DeserializeOwned + serde::Serialize + Identifiable> {
    region:          String,
    queue_table:     String,
    queue_index:     String,
    match_table:     String,
    lock_table:      String,
    lock_regions:    Vec<String>,
    dynamo_client:   dynamodb::Client,
    lock_clients:    Vec<dynamodb::Client>,
    alarm_client:    cloudwatch::Client,
    alarms:          Vec<String>,
    queue_item_type: std::marker::PhantomData<QueueItem>,
}

#[async_trait]
pub trait Processor<QueueItem: serde::de::DeserializeOwned + serde::Serialize + Identifiable> {
    fn client(&self) -> &Client<QueueItem>;
    async fn make_matches(&self, items: Vec<QueueItem>) -> Result<Vec<Vec<QueueItem>>, Error>;
}

pub trait Identifiable {
    fn id(&self) -> String;
}

const CONCURRENT_REQUESTS: usize = 4;
static AWS_CONFIG: once_cell::sync::OnceCell<aws_config::SdkConfig> = once_cell::sync::OnceCell::new();
static LOCK_REGIONS: once_cell::sync::OnceCell<String> = once_cell::sync::OnceCell::new();
static ALARM_NAMES: once_cell::sync::OnceCell<String> = once_cell::sync::OnceCell::new();

pub async fn init() {
    tracing_subscriber::fmt()
        .with_max_level(tracing::Level::WARN)
        // disable printing the name of the module in every log line.
        .with_target(false)
        // disabling time is handy because CloudWatch will add the ingestion time.
        .without_time()
        .init();

    let static_config = aws_config::load_from_env().await;
    AWS_CONFIG.set(static_config).expect("could not save aws config");
    let static_lock_regions = std::env::var("LOCK_REGIONS").expect("LOCK_REGIONS not set");
    LOCK_REGIONS.set(static_lock_regions).expect("could not save lock regions");
    let static_alarm_names = std::env::var("ALARM_NAMES").expect("ALARM_NAMES not set");
    ALARM_NAMES.set(static_alarm_names).expect("could not save alarm names");
}

pub fn match_id<QueueItem: Identifiable>(players: &Vec<QueueItem>) -> String {
    let mut ids: Vec<String> = players.iter().map(|player| player.id()).collect();
    ids.sort();
    ids.join("::")
}

pub async fn handler<QueueItem: serde::de::DeserializeOwned + serde::Serialize + Identifiable>(p: &dyn Processor<QueueItem>) -> Result<(), Error> {
    let handler_func_closure = move |_event: LambdaEvent<Event>| async move {
        run_wrapper(p).await
    };

    lambda_runtime::run(service_fn(handler_func_closure)).await?;
    Ok(())
}

async fn run_wrapper<QueueItem: serde::de::DeserializeOwned + serde::Serialize + Identifiable>(p: &dyn Processor<QueueItem>) -> Result<(), Error> {
    let execution_id = Ksuid::new(None, None);

    match run(p, execution_id).await {
        Ok(_) => Ok(()),
        Err(e) => {
            println!("error: {:?}", e);
            Err("failure")?
        }
    }
}

async fn run<QueueItem: serde::de::DeserializeOwned + serde::Serialize + Identifiable>(p: &dyn Processor<QueueItem>, execution_id: Ksuid) -> Result<(), Error> {
    let client = p.client();

    // first, check if the region is healthy
    // if it is not, make no attempt to process the stream (assuming stream will be processed by other regions)
    match client.check_health().await {
        Ok(healthy) => {
            if !healthy {
                println!("region not healthy, abandoning stream");
                return Ok(())
            }
            println!("region is healthy");
            ()
        }
        Err(e) => {
            println!("failed to check health");
            return Err(e);
        }
    };

    // next, obtain lock, making sure we maintain the index into the regional lock table list
    let (lock_obtained, lock_region_index) = match client.obtain_lock(execution_id).await {
        Ok((obtained, index)) => (obtained, index),
        Err(e) => {
            println!("failed to acquire lock");
            return Err(e);
        }
    };

    // check if current region holds the lock (true by default if it just obtained it)
    let lock_held = match lock_obtained {
        true => {
            println!("lock obtained");
            true
        }
        false => {
            println!("lock not obtained, checking if lock already held");
            match client.check_lock_held(lock_region_index).await {
                Ok(held) => held,
                Err(e) => {
                    println!("failed to check if lock is held");
                    return Err(e);
                }
            }
        }
    };

    // exit if lock not held
    match lock_held {
        true => {
            println!("lock held, processing queue");
            //return Err(Box::new(std::io::Error::new(std::io::ErrorKind::Other, "force error for testing")));
        }
        false => {
            println!("lock not held, abandoning stream");
            // todo: consider custom error so that wrapper can distinguish this from an actual success
            return Ok(());
        }
    };

    // read the queue
    let items = match client.load_queue().await {
        Ok(list) => list,
        Err(e) => {
            println!("failed to read queue");
            return Err(e);
        }
    };

    if items.len() == 0 {
        println!("queue is empty");
        return Ok(());
    } else if items.len() == 1 {
        println!("queue has a single entry, cannot make a match");
        return Ok(());
    }

    // make matches
    let matches = match p.make_matches(items).await {
        Ok(list) => list,
        Err(e) => {
            println!("failed to make matches");
            return Err(e);
        }
    };

    // publish matches
    let publish_results = stream::iter(matches).map(|players| {
        async move {
            client.publish_match(execution_id, players).await
        }
    }).buffer_unordered(CONCURRENT_REQUESTS);

    // wait on results
    let any_match_success = publish_results.any(|result| async {
        match result {
            Ok(id) => {
                println!("published match: {}", id);
                true
            },
            Err(e) => {
                println!("failed to publish match: {:?}", e);
                false
            }
        }
    }).await;

    // todo: inc wait count on unmatched

    // if everything failed, our region ain't healthy
    match any_match_success {
        true => Ok(()),
        false => Err(Box::new(std::io::Error::new(std::io::ErrorKind::Other, "all matches failed to publish")))
    }
}

impl<QueueItem: serde::de::DeserializeOwned + serde::Serialize + Identifiable> Client<QueueItem> {
    pub async fn new() -> Result<Self, Error> {
        let config = AWS_CONFIG.get().expect("No AWS config");
        let region = config.region().expect("No region in config");
        let dynamo_client = dynamodb::Client::new(&config);
        let alarm_client = cloudwatch::Client::new(&config);

        let alarms: Vec<String> = ALARM_NAMES.get().expect("No alarm names").split(",").map(str::to_string).collect();
        let lock_regions: Vec<String> = LOCK_REGIONS.get().expect("No lock regions").split(",").map(str::to_string).collect();
        let mut lock_clients: Vec<dynamodb::Client> = Vec::new();
        for region in lock_regions.clone() {
            let regional_config = dynamodb::config::Builder::from(config)
                .region(aws_types::region::Region::new(region))
                .build();
            lock_clients.push(dynamodb::Client::from_conf(regional_config));
        }

        let queue_table = std::env::var("QUEUE_TABLE").expect("QUEUE_TABLE not set");
        let queue_index = std::env::var("QUEUE_INDEX").expect("QUEUE_INDEX not set");
        let match_table = std::env::var("MATCH_TABLE").expect("MATCH_TABLE not set");
        let lock_table = std::env::var("LOCK_TABLE").expect("LOCK_TABLE not set");

        Ok(Self {
            region: region.to_string(),
            queue_table,
            queue_index,
            match_table,
            lock_table,
            lock_regions,
            dynamo_client,
            lock_clients,
            alarm_client,
            alarms,
            queue_item_type: std::marker::PhantomData,
        })
    }

    async fn check_health(&self) -> Result<bool, Error> {
        let status_result = self.alarm_client.describe_alarms()
            .set_alarm_names(Some(self.alarms.clone()))
            .state_value(cloudwatch::model::StateValue::Alarm)
            .send().await?;

        match status_result.metric_alarms() {
            // we may have an active alarm
            Some(alarms) => match alarms.len() > 0 {
                // we do have an active alarm
                true => {
                    println!("There are active alarms");
                    for alarm in alarms.iter() {
                        println!("Active alarm: {:?}", alarm.alarm_name());
                    };
                    Ok(false)
                },
                // we're good
                false => Ok(true)
            },
            // no alarm currently active
            None => Ok(true),
        }
    }

    async fn obtain_lock(&self, execution_id: Ksuid) -> Result<(bool, usize), Error> {
        for (index, lock_client) in self.lock_clients.iter().enumerate() {
            let region = self.lock_regions.iter().nth(index).expect("invalid region index");
            // todo: time values should be closely related to function timeout, maybe configurable via env var?
            let put_result = lock_client.put_item()
                .table_name(&self.lock_table)
                .item("process", dynamodb::model::AttributeValue::S(format!("queue#{}", self.queue_table)))
                .item("sk", dynamodb::model::AttributeValue::S("lock".to_string()))
                .item("region", dynamodb::model::AttributeValue::S(self.region.clone()))
                .item("ttl", dynamodb::model::AttributeValue::N((execution_id.timestamp_seconds() + 90).to_string()))
                .condition_expression("attribute_not_exists(#ttl) or #ttl < :now_minus_timeout")
                .expression_attribute_names("#ttl", "ttl")
                .expression_attribute_values(":now_minus_timeout", dynamodb::model::AttributeValue::N((execution_id.timestamp_seconds() - 30).to_string()))
                .send().await;

            match put_result {
                Ok(_) => return Ok((true, index)),
                Err(error) => match error {
                    dynamodb::types::SdkError::ServiceError{err, raw: _} => match err.kind {
                        dynamodb::error::PutItemErrorKind::ConditionalCheckFailedException(_) => return Ok((false, index)),
                        _ => {
                            println!("Put lock failed - {} - dynamo sdk error - {:?}", region, err);
                            continue
                        }
                    }
                    other_error => {
                        println!("Put lock failed - {} - unknown error - {:?}", region, other_error);
                        continue
                    }
                }
            }
        }
        Err(Box::new(std::io::Error::new(std::io::ErrorKind::Other, "all regions failed")))
    }

    async fn check_lock_held(&self, lock_region_index: usize) -> Result<bool, Error> {
        let lock_client =  self.lock_clients.iter().nth(lock_region_index).expect("no matching lock client");
        let get_result = lock_client.get_item()
            .table_name(&self.lock_table)
            .key("process", dynamodb::model::AttributeValue::S(format!("queue#{}", self.queue_table)))
            .key("sk", dynamodb::model::AttributeValue::S("lock".to_string()))
            .consistent_read(true)
            .send().await;

        match get_result {
            Ok(get_output) => match get_output.item() {
                Some(lock) => match lock.get("region") {
                    Some(region_av) => match region_av {
                        aws_sdk_dynamodb::model::AttributeValue::S(region) => {
                            println!("lock held by {region}");
                            Ok(region == &self.region)
                        }
                        _ => {
                            println!("invalid region av");
                            Ok(false)
                        }
                    }
                    None => {
                        println!("no region in lock");
                        Ok(false)
                    }
                }
                None => {
                    println!("no lock found");
                    Ok(false)
                }
            }
            Err(e) => Err(Box::new(e))
        }
    }

    async fn load_queue(&self) -> Result<Vec<QueueItem>, Error> {
        let query_result = self.dynamo_client.query()
            .table_name(&self.queue_table)
            .index_name(&self.queue_index)
            .select(dynamodb::model::Select::AllAttributes)
            .key_condition_expression("#queue = :this")
            .expression_attribute_names("#queue", "queue")
            .expression_attribute_values(":this", dynamodb::model::AttributeValue::S(self.queue_table.clone()))
            .send().await;

        match query_result {
            Ok(query_output) => match query_output.items {
                Some(items) => match serde_dynamo::from_items(items) {
                    Ok(parsed_items) => Ok(parsed_items),
                    Err(e) => Err(Box::new(e))
                },
                None => Ok(vec![])
            },
            Err(e) => {
                println!("failed to query queue");
                Err(Box::new(e))
            }
        }
    }

    async fn publish_match(&self, execution_id: Ksuid, players: Vec<QueueItem>) -> Result<String, Error> {
        let id = match_id(&players);
        let players_av = match serde_dynamo::to_attribute_value(players) {
            Ok(av) => av,
            Err(e) => {
                println!("failed to turn players to AV");
                return Err(Box::new(e));
            }
        };

        // todo: delete players from queue (condition check on "state" to make sure player hasn't joined/requeued since)
        let transact_result = self.dynamo_client.transact_write_items()
            .transact_items(
                dynamodb::model::TransactWriteItem::builder()
                    .put(
                        dynamodb::model::Put::builder()
                            .table_name(&self.match_table)
                            .item("match", dynamodb::model::AttributeValue::S(id.clone()))
                            .item("sessionId", dynamodb::model::AttributeValue::S(execution_id.to_base62()))
                            .item("timestamp", dynamodb::model::AttributeValue::N(execution_id.timestamp_seconds().to_string()))
                            .item("ttl", dynamodb::model::AttributeValue::N((execution_id.timestamp_seconds() + 3600).to_string()))
                            .item("queue", dynamodb::model::AttributeValue::S(self.queue_table.clone()))
                            .item("players", players_av)
                            .build()
                    )
                    .build()
            )
            .send().await;

        match transact_result {
            Ok(_) => return Ok(id),
            Err(e) => Err(Box::new(e))
        }
    }
}