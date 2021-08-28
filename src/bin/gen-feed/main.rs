use aws_sdk_s3::ByteStream;
use chrono::{Duration, Utc};
use lambda_runtime::{Context, Error};
use log::{debug, info};
use rss::ItemBuilder;
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::env;

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

    let now = Utc::now();

    info!("Generating feed for date {} ...", now);

    let items: Vec<_> = (0..30)
        .map(|i| now - Duration::days(i))
        .map(|dt| {
            let url = format!(
                "https://{}.s3.amazonaws.com/{}/{}.gif",
                bucket_name,
                strips_dir,
                dt.naive_utc().date(),
            );
            ItemBuilder::default()
                .title(format!("Dilbert - {}", dt.naive_utc().date())) // FIXME
                .link(url.to_owned())
                .description(format!(r#"<img src="{}">"#, url))
                .guid(rss::GuidBuilder::default().value(url).build().unwrap())
                .pub_date(dt.to_rfc2822())
                .build()
                .unwrap() // FIXME
        })
        .collect();

    debug!("{:#?}", &items);

    let channel = rss::ChannelBuilder::default()
        .title("Dilbert")
        .link("https://dilbert.com")
        .description("Dilbert Daily Strip")
        .items(items)
        .build()?;

    let xml = channel.to_string();

    info!("Uploading feed to bucket {} with path {} ...", bucket_name, feed_path);

    let _ = aws_sdk_s3::Client::from_env()
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

// struct FeedBuilder {}

// impl Default for FeedBuilder {
//     fn default() -> Self {
//         FeedBuilder {}
//     }
// }

// impl FeedBuilder {
//     pub fn build() {}
// }
