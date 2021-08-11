use lambda_runtime::{handler_fn, Context, Error};
use log::info;
use reqwest::header::USER_AGENT;
use serde_json::Value;
use std::env;

#[tokio::main]
async fn main() -> Result<(), Error> {
    simple_logger::init_with_level(log::Level::Info)?;
    lambda_runtime::run(handler_fn(handler)).await?;
    Ok(())
}

async fn handler(_: Value, _: Context) -> Result<(), Error> {
    let ep = env::var("HEARTBEAT_ENDPOINT").expect("HEARTBEAT_ENDPOINT not found");

    info!("Sending ping to {}", ep);

    reqwest::Client::new()
        .get(ep)
        .header(USER_AGENT, "dilbert-feed")
        .send()
        .await?
        .error_for_status()?;

    Ok(())
}
