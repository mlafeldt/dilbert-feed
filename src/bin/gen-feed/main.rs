#![deny(clippy::all, clippy::nursery)]
#![deny(nonstandard_style, rust_2018_idioms)]

use std::env;

use anyhow::{Context, Result};
use aws_sdk_s3::{types::ByteStream, Client};
use chrono::Utc;
use lambda_runtime::{service_fn, Error, LambdaEvent};
use log::{debug, info};
use serde::{Deserialize, Serialize};
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
    env_logger::try_init()?;

    let h = Handler {
        bucket_name: &env::var("BUCKET_NAME").expect("BUCKET_NAME not found"),
        strips_dir: &env::var("STRIPS_DIR").expect("STRIPS_DIR not found"),
        feed_path: &env::var("FEED_PATH").expect("FEED_PATH not found"),
        s3_client: Client::new(&aws_config::load_from_env().await),
    };
    debug!("{:?}", h);

    lambda_runtime::run(service_fn(|_: LambdaEvent<Input>| h.handle())).await
}

impl<'a> Handler<'a> {
    async fn handle(&'a self) -> Result<Output> {
        let today = Utc::today().naive_utc();
        let body = self.generate_feed(today).await?;
        let url = self.upload_feed(body).await?;

        Ok(Output { feed_url: url })
    }

    async fn generate_feed(&'a self, date: chrono::NaiveDate) -> Result<ByteStream> {
        info!("Generating feed for date {} ...", date);

        FeedBuilder::default()
            .bucket_name(self.bucket_name)
            .strips_dir(self.strips_dir)
            .start_date(date)
            .s3_client(self.s3_client.clone())
            .build()?
            .xml()
            .await
            .map(|xml| ByteStream::from(xml.into_bytes()))
    }

    async fn upload_feed(&'a self, body: ByteStream) -> Result<Url> {
        info!(
            "Uploading feed to bucket {} with path {} ...",
            self.bucket_name, self.feed_path
        );

        self.s3_client
            .put_object()
            .bucket(self.bucket_name)
            .key(self.feed_path)
            .body(body)
            .content_type("text/xml; charset=utf-8")
            .send()
            .await
            .with_context(|| format!("failed to put object {}", self.feed_path))?;

        let feed_url = format!("https://{}.s3.amazonaws.com/{}", self.bucket_name, self.feed_path).parse()?;

        info!("Upload completed: {}", feed_url);

        Ok(feed_url)
    }
}
