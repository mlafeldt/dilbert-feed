use lambda_runtime::{handler_fn, Context, Error};
use log::info;
use serde_json::Value;
use std::env;

#[tokio::main]
async fn main() -> Result<(), Error> {
    simple_logger::init_with_level(log::Level::Debug)?;
    lambda_runtime::run(handler_fn(handler)).await?;
    Ok(())
}

// Return the input event as output
async fn handler(event: Value, _: Context) -> Result<Value, Error> {
    info!("Hello from {}!", env::var("AWS_LAMBDA_FUNCTION_NAME")?);
    Ok(event)
}
