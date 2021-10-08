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

#[derive(Debug)]
struct Handler<'a> {
    bucket_name: &'a str,
    strips_dir: &'a str,
    http_client: reqwest::Client,
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    env_logger::try_init()?;

    let h = Handler {
        bucket_name: &env::var("BUCKET_NAME").expect("BUCKET_NAME not found"),
        strips_dir: &env::var("STRIPS_DIR").expect("STRIPS_DIR not found"),
        http_client: reqwest::Client::new(),
    };
    debug!("{:?}", h);

    lambda_runtime::run(handler_fn(|input: Input, _: LambdaContext| async {
        let output = h.handle(input).await.map_err(|e| {
            error!("{:?}", e); // log error chain to CloudWatch
            e
        })?;
        Ok(output) as Result<Output>
    }))
    .await?;

    Ok(())
}

impl<'a> Handler<'a> {
    async fn handle(&'a self, input: Input) -> Result<Output> {
        debug!("{:?}", input);

        let comic = ClientBuilder::default()
            .http_client(self.http_client.clone())
            .build()?
            .scrape_comic(input.date)
            .await?;

        info!("Scraping done: {:?}", comic);
        info!("Downloading strip from {} ...", comic.strip_url);

        let image = self
            .http_client
            .get(&comic.image_url)
            .send()
            .await?
            .error_for_status()?
            .bytes()
            .await?;

        info!("Uploading strip to bucket {} ...", self.bucket_name);

        let key = format!("{}/{}.gif", self.strips_dir, comic.date);

        Client::new(&aws_config::load_from_env().await)
            .put_object()
            .bucket(self.bucket_name)
            .key(&key)
            .body(ByteStream::from(image))
            .content_type("image/gif")
            .metadata("title", &comic.title)
            .send()
            .await
            .with_context(|| format!("failed to put object {}", &key))?;

        let upload_url = format!("https://{}.s3.amazonaws.com/{}", self.bucket_name, key);

        info!("Upload completed: {}", upload_url);

        Ok(Output { comic, upload_url })
    }
}
