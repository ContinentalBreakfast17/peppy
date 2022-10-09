use aws_sig_auth::signer::{OperationSigningConfig, RequestConfig, SigV4Signer};
use aws_smithy_http::body::SdkBody;
use aws_smithy_http::byte_stream::ByteStream;
use aws_types::credentials::ProvideCredentials;
use aws_types::region::{Region, SigningRegion};
use aws_types::SigningService;
use futures_util::sink::SinkExt;
use futures_util::StreamExt;
use serde::{Serialize, Deserialize};
use serde_json::json;
use tungstenite::protocol::Message;
use std::collections::HashMap;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use url::{Url, Host};

pub struct Client {
    region:     Region,
    signer:     SigV4Signer,
    config:     aws_config::SdkConfig,
    api:        Url,
    host:       String,
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

#[derive(Debug, Clone, Serialize, Deserialize)]
struct WsMessage {
    #[serde(rename="type")]
    kind: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    payload: Option<serde_json::Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    id: Option<String>,
}

impl Client {
    pub async fn new(config: aws_config::SdkConfig, region: Region, api_url: String) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let signer = aws_sig_auth::signer::SigV4Signer::new();
        let api_client = reqwest::Client::new();
        let api = url::Url::parse(&api_url)?;
        let host = match api.host() {
            Some(host) => match host {
                Host::Domain(domain) => domain.to_string(),
                _ => return Err("invalid api host")?
            },
            None => return Err("inavlid api host")?
        };
        Ok(Self { region: region, signer, config, api, host, api_client })
    }

    pub async fn query<Data: serde::de::DeserializeOwned, Vars: serde::Serialize>(&self, gql_req: GraphqlRequest<Vars>) -> Result<GraphqlResponse<Data>, Box<dyn std::error::Error + Send + Sync>> {
        let body = json!(gql_req).to_string();
        let sdk_body = SdkBody::from(body);

        let mut request = http::Request::builder()
            .method("POST")
            .uri(self.api.as_str())
            .body(sdk_body)?;

        self.sign_request(&mut request).await?;

        let reqw = self.convert_req(request)?;
        let response = self.api_client.execute(reqw)
            .await?
            .json::<GraphqlResponse<Data>>()
            .await?;

        Ok(response)
    }

    // ref: https://docs.aws.amazon.com/appsync/latest/devguide/real-time-websocket-client.html#handshake-details-to-establish-the-websocket-connection
    pub async fn subscribe<Data, Vars, Process, Fut, E>(
        &self,
        gql_req: GraphqlRequest<Vars>,
        process_message: Process,
        start_timeout: u64,
        message_timeout: u64,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>>
    where
        Data: serde::de::DeserializeOwned,
        Vars: serde::Serialize,
        Process: Fn(GraphqlResponse<Data>) -> Fut,
        Fut: futures::Future<Output = Result<Option<()>, E>>,
        E: std::fmt::Debug,
    {
        // form request
        let id = uuid::Uuid::new_v4().to_string();
        let connect_headers = self.ws_header("{}".to_string(), "/graphql/connect".to_string()).await?;
        let encoded_connect_header = base64::encode_config(json!(connect_headers).to_string().as_bytes(), base64::URL_SAFE);
        let request = http::Request::builder()
            .method("GET")
            //.uri(format!("{}?header={}&payload=e30=", self.api.as_str(), header))
            .uri(format!("wss://{}/graphql/realtime?header={}&payload=e30=", self.host, encoded_connect_header))
            .header("Host", self.host.clone())
            .header("Upgrade", "websocket")
            .header("Connection", "Upgrade")
            .header("Cache-Control", "no-cache")
            .header("Sec-WebSocket-Version", "13")
            .header("Sec-WebSocket-Protocol", "graphql-ws")
            .header("Sec-WebSocket-Extensions", "permessage-deflate; client_max_window_bits")
            .header("Sec-WebSocket-Key", tungstenite::handshake::client::generate_key())
            .body(())?;

        // dial
        let (mut ws_stream, _) = tokio_tungstenite::connect_async(request).await?;
        let ms_stream_ref = &mut ws_stream;

        // init connection
        ms_stream_ref.send(Message::Text(json!(WsMessage{
            kind: "connection_init".to_string(),
            payload: None,
            id: None,
        }).to_string())).await?;

        let last_ka = &mut current_time_ms()?;
        let connection_timeout_ms = match self.ws_poll_msg(
            ms_stream_ref,
            "connection_ack".to_string(),
            last_ka,
            1001, // this value doesn't matter since we haven't initialized yet, just need to be more than the timeout
            1000,
        ).await {
            Ok(msg) => match msg.payload {
                Some(map) => match map.get("connectionTimeoutMs") {
                    Some(connection_timeout_ms_val) => match connection_timeout_ms_val {
                        serde_json::Value::Number(number) => match number.as_u64() {
                            Some(n) => u128::from(n),
                            None => return Err("could not get init ack: connectionTimeoutMs not convertible to u64")?
                        },
                        _ => return Err("could not get init ack: connectionTimeoutMs not a number")?
                    },
                    None => return Err("could not get init ack: connectionTimeoutMs not in payload")?
                },
                None => return Err("could not get init ack: got no payload")?
            },
            Err(e) => return Err(format!("could not get init ack: {:?}", e))?
        };

        // subscribe
        let sub_body = json!(gql_req).to_string();
        let sub_headers = self.ws_header(sub_body.clone(), "/graphql".to_string()).await?;
        let sub_payload = json!({
            "data": sub_body,
            "extensions": {
                "authorization": sub_headers,
            },
        });
        let sub_msg = WsMessage{kind: "start".to_string(), id: Some(id.clone()), payload: Some(sub_payload)};
        ms_stream_ref.send(Message::Text(json!(sub_msg).to_string())).await?;

        match self.ws_poll_msg(
            ms_stream_ref,
            "start_ack".to_string(),
            last_ka,
            connection_timeout_ms,
            start_timeout,
        ).await {
            Ok(_) => println!("subscription started"),
            Err(e) => return Err(format!("could not start sub: {:?}", e))?
        };

        // process
        loop {
            let response = match self.ws_poll_msg(
                ms_stream_ref,
                "data".to_string(),
                last_ka,
                connection_timeout_ms,
                message_timeout,
            ).await {
                Ok(msg) => match msg.payload {
                    Some(json_value) => match serde_json::from_value::<GraphqlResponse<Data>>(json_value) {
                        Ok(response) => response,
                        Err(e) => return Err(format!("subscription failed: invalid json: {:?}", e))?
                    },
                    None => return Err("subscription failed: no payload")?
                },
                Err(e) => return Err(format!("subscription failed: {:?}", e))?
            };

            match process_message(response).await {
                Ok(result) => match result {
                    Some(_) => (),
                    None => break,
                },
                Err(e) => return Err(format!("subscription failed: process message failure: {:?}", e))?
            };
        }

        // stop
        let stop_msg = WsMessage{kind: "stop".to_string(), id: Some(id.clone()), payload: None};
        ms_stream_ref.send(Message::Text(json!(stop_msg).to_string())).await?;
        match self.ws_poll_msg(
            ms_stream_ref,
            "complete".to_string(),
            last_ka,
            connection_timeout_ms,
            1000,
        ).await {
            Ok(_) => println!("subscription stopped"),
            Err(e) => return Err(format!("could not stop sub: {:?}", e))?
        };

        // close connection
        match ms_stream_ref.close(None).await {
            Ok(_) => (),
            Err(e) => println!("failed to close ws: {:?}", e)
        };
        Ok(())
    }

    async fn ws_poll_msg(
        &self,
        ws_stream: &mut tokio_tungstenite::WebSocketStream<tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>>,
        kind: String,
        last_ka: &mut u128,
        connection_timeout_ms: u128,
        max_wait_ms: u64,
    ) -> Result<WsMessage, Box<dyn std::error::Error + Send + Sync>> {
        let entry_time = current_time_ms()?;
        while max_wait_ms == 0 || entry_time + u128::from(max_wait_ms) >= current_time_ms()? {
            match tokio::time::timeout(Duration::from_millis(max_wait_ms), ws_stream.next()).await {
                Ok(next) => match next {
                    Some(result) => match result {
                        Ok(message) => match message {
                            Message::Text(text) => match serde_json::from_str::<WsMessage>(&text) {
                                Ok(ws_msg) => {
                                    if ws_msg.kind == kind {
                                        return Ok(ws_msg);
                                    } else if ws_msg.kind == "ka".to_string() {
                                        println!("received ka");
                                        *last_ka = current_time_ms()?;
                                        continue;
                                    }

                                    return Err(format!("unexpected msg type: {:?}", ws_msg))?;
                                },
                                Err(e) => return Err(format!("invalid text msg: {:?}", e))?
                            },
                            // throw out other messages
                            _ => {
                                println!("received non text msg");
                                continue
                            }
                        },
                        Err(e) => return Err(format!("receive failure: {:?}", e))?
                    },
                    None => return Err("connection closed")?
                },
                Err(_) => Err("timed out")?
            };

            if *last_ka + connection_timeout_ms < current_time_ms()? {
                return Err("lost connection")?;
            }
        }

        Err("timed out")?
    }

    async fn ws_header(&self, body: String, path: String) -> Result<HashMap<String, String>, Box<dyn std::error::Error + Send + Sync>> {
        // sign appsync request
        let sdk_body = SdkBody::from(body);
        let mut sdk_request = http::Request::builder()
            .method("POST")
            .uri(format!("https://{}{}", self.host, path))
            .header("accept", "application/json, text/javascript")
            .header("content-encoding", "amz-1.0")
            .header("content-type", "application/json; charset=UTF-8")
            .body(sdk_body)?;

        self.sign_request(&mut sdk_request).await?;

        // build "header" param
        let mut header: HashMap<String, String> = HashMap::new();
        header.insert(
            "accept".to_string(),
            sdk_request.headers().get("accept").unwrap().to_str().unwrap().to_string(),
        );
        header.insert(
            "content-encoding".to_string(),
            sdk_request.headers().get("content-encoding").unwrap().to_str().unwrap().to_string(),
        );
        header.insert(
            "content-type".to_string(),
            sdk_request.headers().get("content-type").unwrap().to_str().unwrap().to_string(),
        );
        header.insert(
            "x-amz-date".to_string(),
            sdk_request.headers().get("x-amz-date").unwrap().to_str().unwrap().to_string(),
        );
        header.insert(
            "Authorization".to_string(),
            sdk_request.headers().get("authorization").unwrap().to_str().unwrap().to_string(),
        );
        header.insert(
            "host".to_string(),
            self.host.clone(),
        );

        // add session token if contained in creds
        match self.config.credentials_provider() {
            Some(p) => match p.provide_credentials().await {
                Ok(creds) => match creds.session_token() {
                    Some(token) => header.insert(
                        "X-Amz-Security-Token".to_string(),
                        token.to_string(),
                    ),
                    None => None
                },
                Err(e) => return Err(e)?
            }
            None => return Err("failed to get credentials provider")?
        };

        Ok(header)
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

fn current_time_ms() -> Result<u128, Box<dyn std::error::Error + Send + Sync>> {
    let now = SystemTime::now();
    match now.duration_since(UNIX_EPOCH) {
        Ok(epoch) => Ok(epoch.as_millis()),
        Err(e) => Err(format!("could not get epoch time: {:?}", e))?
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Healthcheck {
    id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct HealthcheckResponse {
    #[serde(rename = "healthcheck")]
    healthcheck: Option<Healthcheck>,
}

const HEALTCHECK_QUERY: &str = "
subscription($id: ID!){
    healthcheck(id: $id) {
        id
    }
}
";

#[allow(dead_code)]
async fn test() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    let config = aws_config::load_from_env().await;
    let region = config.region().ok_or("No region in config")?.clone();
    let api = std::env::var("API_URL")?;
    let client = Client::new(config, region.clone(), api).await?;

    let req = GraphqlRequest{
        query: HEALTCHECK_QUERY.to_string(),
        variables: Healthcheck{id: uuid::Uuid::new_v4().to_string()}
    };
    client.subscribe(req, test_process, 2000, 10000).await
}

async fn test_process(response: GraphqlResponse<HealthcheckResponse>) -> Result<Option<()>, Box<dyn std::error::Error + Send + Sync>> {
    println!("{:?}", response);
    Ok(None)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn it_works() {
        match test().await {
            Err(e) => panic!("error: {:?}", e),
            Ok(_) => ()
        };
    }
}
