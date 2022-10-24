use std::collections::HashMap;
use std::env;

use anyhow::{bail, Result};
use lambda_runtime::{service_fn, Error, LambdaEvent};
use reqwest::{redirect, Client};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use tracing::{info, instrument};

#[derive(Deserialize, Debug)]
struct Input {
    endpoint: Option<String>,

    #[allow(dead_code)]
    #[serde(flatten)]
    extra: HashMap<String, Value>,
}

#[derive(Serialize, PartialEq, Debug)]
struct Output {
    endpoint: String,
    status: u16,
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    tracing_subscriber::fmt()
        .with_max_level(tracing_subscriber::filter::LevelFilter::INFO)
        .compact()
        .try_init()?;

    let http_client = Client::builder()
        .user_agent("dilbert-feed")
        .redirect(redirect::Policy::none())
        .build()?;

    lambda_runtime::run(service_fn(|input: LambdaEvent<Input>| {
        handler(input.payload, http_client.clone())
    }))
    .await
}

#[instrument(ret)]
async fn handler(input: Input, http_client: Client) -> Result<Output> {
    let ep = input
        .endpoint
        .unwrap_or_else(|| env::var("HEARTBEAT_ENDPOINT").expect("HEARTBEAT_ENDPOINT not found"));

    info!("Sending ping to {ep}");

    let resp = http_client.get(&ep).send().await?;

    if !resp.status().is_success() {
        bail!("HTTP status not 2xx: {}", resp.status());
    }

    Ok(Output {
        endpoint: ep,
        status: resp.status().as_u16(),
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use pretty_assertions::assert_eq;
    use wiremock::{
        matchers::method,
        {Mock, MockServer, ResponseTemplate},
    };

    #[tokio::test]
    async fn test_handler_200() {
        let server = MockServer::start().await;

        Mock::given(method("GET"))
            .respond_with(ResponseTemplate::new(200))
            .expect(1)
            .mount(&server)
            .await;

        let resp = handler(
            Input {
                endpoint: Some(server.uri()),
                extra: Default::default(),
            },
            Default::default(),
        )
        .await
        .unwrap();

        assert_eq!(
            resp,
            Output {
                endpoint: server.uri(),
                status: 200,
            },
        );
    }

    #[tokio::test]
    #[should_panic(expected = "HTTP status not 2xx: 503 Service Unavailable")]
    async fn test_handler_503() {
        let server = MockServer::start().await;

        Mock::given(method("GET"))
            .respond_with(ResponseTemplate::new(503))
            .expect(1)
            .mount(&server)
            .await;

        handler(
            Input {
                endpoint: Some(server.uri()),
                extra: Default::default(),
            },
            Default::default(),
        )
        .await
        .unwrap();
    }
}
