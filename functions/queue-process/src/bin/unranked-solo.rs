use lambda_runtime::{Error};
use async_trait::async_trait;
use serde::{Serialize, Deserialize};
use geo::{EuclideanDistance, point};
use queue_processor::{Client, init, handler};

#[derive(Debug, Serialize, Deserialize)]
struct QueueItem {
    user: String,
    ip: String,
    mmr: i64,
    join_time: i64,
    coordinates: Coordinates,
    #[serde(default)]
    wait_count: i64,
}

#[derive(Debug, Serialize, Deserialize)]
struct Coordinates {
    latitude: f64,
    longitude: f64,
}

struct Processor {
    client: Client<QueueItem>,
}

impl Processor {
    async fn new() -> Result<Self, Error> {
        match Client::new().await {
            Ok(client) => Ok(Self { client }),
            Err(e) => Err(e)
        }
    }

    // current implementation naively checks if mmr is within a certain threshold
    fn compare_mmr(&self, a: &QueueItem, b: &QueueItem) -> i64 {
        if (a.mmr - b.mmr).abs() < 100 {
            return 0;
        }
        return 1;
    }

    fn compare_distance(&self, a: &QueueItem, b: &QueueItem) -> i64 {
        let a_point = point!(x: a.coordinates.longitude, y: a.coordinates.latitude);
        let b_point = point!(x: b.coordinates.longitude, y: b.coordinates.latitude);

        if (a_point.euclidean_distance(&b_point)).abs() < 100.0 {
            return 0;
        }
        return 1;
    }

    fn compare_ip(&self, a: &QueueItem, b: &QueueItem) -> i64 {
        if a.ip != b.ip {
            return 0;
        }
        return 10000;
    }
}

#[async_trait]
impl queue_processor::Processor<QueueItem> for Processor {
    fn client(&self) -> &Client<QueueItem> {
        &self.client
    }

    async fn process_items(&self, items: Vec<QueueItem>) -> Result<(), Error> {
        let mut matches: Vec<(&QueueItem, &QueueItem)> = Vec::new();
        let mut matched_users = std::collections::HashSet::new();

        for (index, a) in items.iter().enumerate() {
            if index >= items.iter().len() - 1 {
                break
            } else if matched_users.contains(&a.user) {
                continue
            }

            for b in items.iter().skip(index + 1) {
                if matched_users.contains(&b.user) {
                    continue
                }

                let score = self.compare_ip(a, b) + self.compare_mmr(a, b) + self.compare_distance(a, b);
                if score == 0 {
                    // compatible
                    matches.push((a, b));
                    matched_users.insert(a.user.clone());
                    matched_users.insert(b.user.clone());
                    break
                }
            }
        }

        for (a, b) in matches.iter() {
            println!("{:?} - {:?}", a.user, b.user);
        }

        Ok(())
    }
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    init().await;
    let processor = Processor::new().await.expect("Failed to init processor");
    handler(&processor).await
}