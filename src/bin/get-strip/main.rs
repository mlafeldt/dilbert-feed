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
    lambda_runtime::run(handler_fn(handler)).await?;
    Ok(())
}

async fn handler(input: Input, _: Context) -> Result<Output, Error> {
    debug!("Got input: {:?}", input);

    let bucket_name = env::var("BUCKET_NAME").expect("BUCKET_NAME not found");
    let strips_dir = env::var("STRIPS_DIR").expect("STRIPS_DIR not found");

    let comic = Dilbert::default().scrape_comic(input.date).await?;

    info!("Scraping done: {:?}", comic);
    info!("Downloading strip from {} ...", comic.strip_url);

    let image = reqwest::get(&comic.image_url)
        .await?
        .error_for_status()?
        .bytes()
        .await?;

    info!("Uploading strip to bucket {} ...", bucket_name);

    let key = format!("{}/{}.gif", strips_dir, comic.date);

    let _ = Client::from_env()
        .put_object()
        .bucket(&bucket_name)
        .key(&key)
        .body(ByteStream::from(image))
        .content_type("image/gif")
        .metadata("title", &comic.title)
        .send()
        .await?;

    let upload_url = format!("https://{}.s3.amazonaws.com/{}", bucket_name, key);

    info!("Upload completed: {}", upload_url);

    Ok(Output { comic, upload_url })
}
