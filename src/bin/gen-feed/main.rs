use aws_sdk_s3::{ByteStream, Client};
use chrono::Utc;
use lambda_runtime::{Context, Error};
use log::info;
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::env;

mod feed;
use feed::FeedBuilder;

#[derive(Deserialize, Debug)]
struct Input {}

#[derive(Serialize, PartialEq, Debug)]
struct Output {
    feed_url: String,
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    simple_logger::init_with_env()?;
    // lambda_runtime::run(handler_fn(handler)).await?;
    info!("{}", json!(handler(Input {}, Context::default()).await?));
    Ok(())
}

async fn handler(_: Input, _: Context) -> Result<Output, Error> {
    let bucket_name = env::var("BUCKET_NAME").expect("BUCKET_NAME not found");
    let strips_dir = env::var("STRIPS_DIR").expect("STRIPS_DIR not found");
    let feed_path = env::var("FEED_PATH").expect("FEED_PATH not found");

    let today = Utc::today().naive_utc();

    info!("Generating feed for date {} ...", today);

    let xml = FeedBuilder::default()
        .bucket_name(&bucket_name)
        .strips_dir(&strips_dir)
        .start_date(today)
        .build()?
        .xml()?;

    info!("Uploading feed to bucket {} with path {} ...", bucket_name, feed_path);

    let _ = Client::from_env()
        .put_object()
        .bucket(&bucket_name)
        .key(&feed_path)
        .body(ByteStream::from(xml.into_bytes()))
        .content_type("text/xml; charset=utf-8")
        .send()
        .await?;

    let feed_url = format!("https://{}.s3.amazonaws.com/{}", bucket_name, feed_path);

    info!("Upload completed: {}", feed_url);

    Ok(Output { feed_url })
}
