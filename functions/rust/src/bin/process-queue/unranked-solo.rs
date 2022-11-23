use async_trait::async_trait;
use geo::{EuclideanDistance, point};
use lambda_runtime::{Error};
use queue_processor::{Client, init, handler, match_id};
use serde::{Serialize, Deserialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
struct QueueItem {
    user: String,
    ip: String,
    region: String,
    mmr: i64,
    join_time: i64,
    coordinates: Coordinates,
    queue: String,
    #[serde(default)]
    wait_count: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Coordinates {
    latitude: f64,
    longitude: f64,
}

impl queue_processor::Identifiable for QueueItem {
    fn id(&self) -> String {
        self.user.clone()
    }
}

impl queue_processor::Identifiable for &QueueItem {
    fn id(&self) -> String {
        self.user.clone()
    }
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

    fn compatible(&self, a: &QueueItem, b: &QueueItem) -> bool {
        let checks = vec![
            self.compatible_mmr(a, b),
            self.compatible_distance(a, b),
            self.compatible_ip(a, b),
            // todo: check recent match history
        ];
        checks.iter().all(|check| *check)
    }

    fn compatible_mmr(&self, a: &QueueItem, b: &QueueItem) -> bool {
        (a.mmr - b.mmr).abs() < 100
    }

    fn compatible_distance(&self, a: &QueueItem, b: &QueueItem) -> bool {
        let a_point = point!(x: a.coordinates.longitude, y: a.coordinates.latitude);
        let b_point = point!(x: b.coordinates.longitude, y: b.coordinates.latitude);

        (a_point.euclidean_distance(&b_point)).abs() < 100.0
    }

    fn compatible_ip(&self, a: &QueueItem, b: &QueueItem) -> bool {
        a.ip != b.ip
    }
}

#[async_trait]
impl queue_processor::Processor<QueueItem> for Processor {
    fn client(&self) -> &Client<QueueItem> {
        &self.client
    }

    async fn make_matches(&self, items: Vec<QueueItem>) -> Result<Vec<Vec<QueueItem>>, Error> {
        let mut graph = petgraph::Graph::<&QueueItem, String, petgraph::Undirected>::default();
        for item in items.iter() {
            graph.add_node(item);
        }

        for (i, node_a_index) in graph.node_indices().enumerate() {
            if i == graph.node_indices().len() - 1 {
                // we're done
                break
            }

            // add an edge to every node after it in the list
            for node_b_index in graph.node_indices().skip(i + 1) {
                let a = graph[node_a_index];
                let b = graph[node_b_index];
                if self.compatible(a, b) {
                    let v = vec![a, b];
                    let edge = match_id(&v);
                    graph.add_edge(node_a_index, node_b_index, edge);
                }
            }
        }

        let mut matches: Vec<Vec<QueueItem>> = Vec::new();
        let matching = petgraph::algo::matching::greedy_matching(&graph);
        for (node_a_index, node_b_index) in matching.edges() {
            let a = graph[node_a_index].clone();
            let b = graph[node_b_index].clone();
            matches.push(vec![a, b]);
        }
        Ok(matches)
    }
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    init().await;
    let processor = Processor::new().await.expect("Failed to init processor");
    handler(&processor).await
}