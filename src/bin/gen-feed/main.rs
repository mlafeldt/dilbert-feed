use std::env;

use anyhow::{Context, Result};
use aws_sdk_s3::{types::ByteStream, Client};
use chrono::Utc;
use lambda_runtime::{service_fn, Error, LambdaEvent};
use serde::{Deserialize, Serialize};
use tracing::{info, instrument};
use url::Url;

mod feed;
use feed::FeedBuilder;

#[derive(Deserialize, Debug)]
struct Input {}

#[derive(Serialize, PartialEq, Debug)]
struct Output {
    feed_url: Url,
}

#[derive(Debug)]
struct Handler<'a> {
    bucket_name: &'a str,
    strips_dir: &'a str,
    feed_path: &'a str,
    s3_client: Client,
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    tracing_subscriber::fmt()
        .with_max_level(tracing_subscriber::filter::LevelFilter::INFO)
        .compact()
        .try_init()?;

    let h = Handler {
        bucket_name: &env::var("BUCKET_NAME").expect("BUCKET_NAME not found"),
        strips_dir: &env::var("STRIPS_DIR").expect("STRIPS_DIR not found"),
        feed_path: &env::var("FEED_PATH").expect("FEED_PATH not found"),
        s3_client: Client::new(&aws_config::load_from_env().await),
    };

    lambda_runtime::run(service_fn(|_: LambdaEvent<Input>| h.handle())).await
}

impl<'a> Handler<'a> {
    #[instrument(ret)]
    async fn handle(&'a self) -> Result<Output> {
        let today = Utc::today().naive_utc();

        info!("Generating feed for date {today} ...");

        let xml = FeedBuilder::default()
            .bucket_name(self.bucket_name)
            .strips_dir(self.strips_dir)
            .start_date(today)
            .s3_client(self.s3_client.clone())
            .build()?
            .xml()
            .await?;

        info!(
            "Uploading feed to bucket {} with path {} ...",
            self.bucket_name, self.feed_path
        );

        self.s3_client
            .put_object()
            .bucket(self.bucket_name)
            .key(self.feed_path)
            .body(ByteStream::from(xml.into_bytes()))
            .content_type("text/xml; charset=utf-8")
            .send()
            .await
            .with_context(|| format!("failed to put object {}", self.feed_path))?;

        let feed_url = format!("https://{}.s3.amazonaws.com/{}", self.bucket_name, self.feed_path).parse()?;

        info!("Upload completed: {feed_url}");

        Ok(Output { feed_url })
    }
}
