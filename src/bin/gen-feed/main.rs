#![deny(clippy::all, clippy::nursery)]
#![deny(nonstandard_style, rust_2018_idioms)]

use anyhow::{Context, Result};
use aws_sdk_s3::{ByteStream, Client};
use chrono::Utc;
use lambda_runtime::{handler_fn, Context as LambdaContext, Error};
use log::{error, info};
use serde::{Deserialize, Serialize};
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
    env_logger::try_init()?;

    lambda_runtime::run(handler_fn(|_: Input, _: LambdaContext| async {
        let output = handler().await.map_err(|e| {
            error!("{:?}", e); // log error chain to CloudWatch
            e
        })?;
        Ok(output) as Result<Output>
    }))
    .await?;

    Ok(())
}

async fn handler() -> Result<Output> {
    let bucket_name = env::var("BUCKET_NAME").expect("BUCKET_NAME not found");
    let strips_dir = env::var("STRIPS_DIR").expect("STRIPS_DIR not found");
    let feed_path = env::var("FEED_PATH").expect("FEED_PATH not found");

    let today = Utc::today().naive_utc();

    info!("Generating feed for date {} ...", today);

    let client = Client::new(&aws_config::load_from_env().await);

    let xml = FeedBuilder::default()
        .bucket_name(&bucket_name)
        .strips_dir(&strips_dir)
        .start_date(today)
        .s3_client(&client)
        .build()?
        .xml()
        .await?;

    info!("Uploading feed to bucket {} with path {} ...", bucket_name, feed_path);

    client
        .put_object()
        .bucket(&bucket_name)
        .key(&feed_path)
        .body(ByteStream::from(xml.into_bytes()))
        .content_type("text/xml; charset=utf-8")
        .send()
        .await
        .with_context(|| format!("failed to put object {}", &feed_path))?;

    let feed_url = format!("https://{}.s3.amazonaws.com/{}", bucket_name, feed_path);

    info!("Upload completed: {}", feed_url);

    Ok(Output { feed_url })
}
