#![deny(clippy::all, clippy::nursery)]
#![deny(nonstandard_style, rust_2018_idioms)]

use anyhow::Result;
use async_trait::async_trait;
use aws_sdk_s3::{ByteStream, Client};
use chrono::NaiveDate;
use lambda_runtime::{handler_fn, Context, Error};
use log::{debug, info};
use serde::{Deserialize, Serialize};
use std::env;

mod dilbert;
use dilbert::{Comic, Dilbert};

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
    simple_logger::init_with_env()?;

    let client = Client::new(&aws_config::load_from_env().await);
    let bucket_name = env::var("BUCKET_NAME").expect("BUCKET_NAME not found");
    let repo = S3Repository::new(&bucket_name, &client);
    // let repo = InMemRepository::default();

    let h = Handler {
        bucket_name: &bucket_name,
        repo: Box::new(repo),
    };

    lambda_runtime::run(handler_fn(|input: Input, ctx: Context| async {
        let output = h.handle(input, ctx).await?;
        Ok::<Output, Error>(output)
    }))
    .await?;

    Ok(())
}

struct Handler<'a> {
    bucket_name: &'a str,
    repo: Box<dyn Repository + Send + Sync + 'a>,
}

impl<'a> Handler<'a> {
    async fn handle(&'a self, input: Input, _: Context) -> Result<Output> {
        debug!("Got input: {:?}", input);

        let strips_dir = env::var("STRIPS_DIR").expect("STRIPS_DIR not found");

        let comic = Dilbert::default().scrape_comic(input.date).await?;

        info!("Scraping done: {:?}", comic);
        info!("Downloading strip from {} ...", comic.strip_url);

        let image = reqwest::get(&comic.image_url)
            .await?
            .error_for_status()?
            .bytes()
            .await?;

        info!("Uploading strip to bucket {} ...", self.bucket_name);

        let key = format!("{}/{}.gif", strips_dir, comic.date);
        let upload_url = self.repo.store(&key, image.to_vec(), &comic.title).await?;

        info!("Upload completed: {}", upload_url);

        Ok(Output { comic, upload_url })
    }
}

#[async_trait]
trait Repository {
    async fn store(&self, key: &str, body: Vec<u8>, title: &str) -> Result<String>;
}

#[derive(Debug)]
struct S3Repository<'a> {
    bucket_name: &'a str,
    s3_client: &'a Client,
}

impl<'a> S3Repository<'a> {
    const fn new(bucket_name: &'a str, s3_client: &'a Client) -> Self {
        Self { bucket_name, s3_client }
    }
}

#[async_trait]
impl Repository for S3Repository<'_> {
    async fn store(&self, key: &str, body: Vec<u8>, title: &str) -> Result<String> {
        self.s3_client
            .put_object()
            .bucket(self.bucket_name)
            .key(key)
            .body(ByteStream::from(body))
            .content_type("image/gif")
            .metadata("title", title)
            .send()
            .await?;

        let upload_url = format!("https://{}.s3.amazonaws.com/{}", self.bucket_name, key);

        Ok(upload_url)
    }
}

#[derive(Default, Debug)]
struct InMemRepository {}

#[async_trait]
impl Repository for InMemRepository {
    async fn store(&self, _key: &str, _body: Vec<u8>, _title: &str) -> Result<String> {
        Ok(String::default())
    }
}
