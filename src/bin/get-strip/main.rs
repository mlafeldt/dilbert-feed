#![deny(clippy::all, clippy::nursery)]
#![deny(nonstandard_style, rust_2018_idioms)]

use std::env;

use anyhow::{Context, Result};
use aws_sdk_s3::types::ByteStream;
use chrono::NaiveDate;
use lambda_runtime::{service_fn, Error, LambdaEvent};
use serde::{Deserialize, Serialize};
use tracing::{info, instrument};
use url::Url;

mod dilbert;
use dilbert::{ClientBuilder, Comic};

#[derive(Deserialize, Debug)]
struct Input {
    date: Option<NaiveDate>,
}

#[derive(Serialize, PartialEq, Debug)]
struct Output {
    #[serde(flatten)]
    comic: Comic,
    upload_url: Url,
}

#[derive(Debug)]
struct Handler<'a> {
    bucket_name: &'a str,
    strips_dir: &'a str,
    http_client: reqwest::Client,
    s3_client: aws_sdk_s3::Client,
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
        http_client: reqwest::Client::new(),
        s3_client: aws_sdk_s3::Client::new(&aws_config::load_from_env().await),
    };

    lambda_runtime::run(service_fn(|input: LambdaEvent<Input>| h.handle(input.payload))).await
}

impl<'a> Handler<'a> {
    #[instrument]
    async fn handle(&'a self, input: Input) -> Result<Output> {
        let comic = ClientBuilder::default()
            .http_client(self.http_client.clone())
            .build()?
            .scrape_comic(input.date)
            .await?;

        info!("Scraping done: {comic:?}");
        info!("Downloading strip from {} ...", comic.strip_url);

        let image = self
            .http_client
            .get(comic.image_url.clone())
            .send()
            .await?
            .error_for_status()?
            .bytes()
            .await?;

        info!("Uploading strip to bucket {} ...", self.bucket_name);

        let key = format!("{}/{}.gif", self.strips_dir, comic.date);

        self.s3_client
            .put_object()
            .bucket(self.bucket_name)
            .key(&key)
            .body(ByteStream::from(image))
            .content_type("image/gif")
            .metadata("title", &comic.title)
            .send()
            .await
            .with_context(|| format!("failed to put object {}", &key))?;

        let upload_url = format!("https://{}.s3.amazonaws.com/{}", self.bucket_name, key).parse()?;

        info!("Upload completed: {upload_url}");

        Ok(Output { comic, upload_url })
    }
}
