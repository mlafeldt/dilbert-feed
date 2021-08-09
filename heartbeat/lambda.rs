use hyper::{Body, Client, Request, Uri};
use lambda_runtime::{handler_fn, Context, Error};
use serde_json::Value;
use std::env;

#[tokio::main]
async fn main() -> Result<(), Error> {
    simple_logger::init_with_level(log::Level::Trace)?;
    lambda_runtime::run(handler_fn(handler)).await?;
    Ok(())
}

async fn handler(_: Value, _: Context) -> Result<(), Error> {
    let endpoint: Uri = env::var("HEARTBEAT_ENDPOINT")
        .expect("heartbeat endpoint must be set")
        .parse()
        .unwrap();

    let req = Request::get(endpoint)
        .header("User-Agent", "dilbert-feed-rust")
        .body(Body::empty())
        .unwrap();

    let resp = Client::new().request(req).await?;
    if !resp.status().is_success() {
        return Err(format!("HTTP error: {}", resp.status()).into());
    };

    Ok(())
}
