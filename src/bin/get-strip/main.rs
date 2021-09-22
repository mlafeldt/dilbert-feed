#![deny(clippy::all, clippy::nursery)]
#![deny(nonstandard_style, rust_2018_idioms)]

use anyhow::{Context, Result};
use aws_sdk_s3::{ByteStream, Client};
use chrono::NaiveDate;
use lambda_runtime::{handler_fn, Context as LambdaContext, Error};
use log::{debug, error, info};
use serde::{Deserialize, Serialize};
use std::env;

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

    upload_url: String,
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    env_logger::try_init()?;

    let http_client = reqwest::Client::new();

    lambda_runtime::run(handler_fn(|input: Input, _: LambdaContext| async {
        let output = handler(input, http_client.clone()).await.map_err(|e| {
            error!("{:?}", e); // log error chain to CloudWatch
            e
        })?;
        Ok(output) as Result<Output>
    }))
    .await?;

    Ok(())
}

async fn handler(input: Input, http_client: reqwest::Client) -> Result<Output> {
    debug!("Got input: {:?}", input);

    let bucket_name = env::var("BUCKET_NAME").expect("BUCKET_NAME not found");
    let strips_dir = env::var("STRIPS_DIR").expect("STRIPS_DIR not found");

    let comic = ClientBuilder::default()
        .http_client(http_client.clone())
        .build()?
        .scrape_comic(input.date)
        .await?;

    info!("Scraping done: {:?}", comic);
    info!("Downloading strip from {} ...", comic.strip_url);

    let image = http_client
        .get(&comic.image_url)
        .send()
        .await?
        .error_for_status()?
        .bytes()
        .await?;

    info!("Uploading strip to bucket {} ...", bucket_name);

    let key = format!("{}/{}.gif", strips_dir, comic.date);

    Client::new(&aws_config::load_from_env().await)
        .put_object()
        .bucket(&bucket_name)
        .key(&key)
        .body(ByteStream::from(image))
        .content_type("image/gif")
        .metadata("title", &comic.title.replace(|c: char| !c.is_ascii(), "?"))
        .send()
        .await
        .with_context(|| format!("failed to put object {}", &key))?;

    let upload_url = format!("https://{}.s3.amazonaws.com/{}", bucket_name, key);

    info!("Upload completed: {}", upload_url);

    Ok(Output { comic, upload_url })
}
