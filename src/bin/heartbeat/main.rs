#![deny(clippy::all, clippy::nursery)]
#![deny(nonstandard_style, rust_2018_idioms)]

use std::collections::HashMap;
use std::env;

use anyhow::{bail, Result};
use futures_util::TryFutureExt;
use lambda_runtime::{handler_fn, Context as LambdaContext, Error};
use log::{debug, error, info};
use reqwest::{redirect, Client};
use serde::{Deserialize, Serialize};
use serde_json::Value;

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
    env_logger::try_init()?;

    let http_client = Client::builder()
        .user_agent("dilbert-feed")
        .redirect(redirect::Policy::none())
        .build()?;

    lambda_runtime::run(handler_fn(|input: Input, _: LambdaContext| {
        handler(input, http_client.clone()).inspect_err(|e| error!("{:?}", e))
    }))
    .await
}

async fn handler(input: Input, http_client: Client) -> Result<Output> {
    debug!("{:?}", input);

    let ep = input
        .endpoint
        .unwrap_or_else(|| env::var("HEARTBEAT_ENDPOINT").expect("HEARTBEAT_ENDPOINT not found"));

    info!("Sending ping to {}", ep);

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
    use wiremock::matchers::method;
    use wiremock::{Mock, MockServer, ResponseTemplate};

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
                extra: HashMap::new(),
            },
            Client::default(),
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
                extra: HashMap::new(),
            },
            Client::default(),
        )
        .await
        .unwrap();
    }
}
