use lambda_runtime::{Context, Error};
use log::{debug, info};
use serde::{Deserialize, Serialize};
use serde_json::json;
// use std::env;

mod dilbert;
use dilbert::{Comic, Dilbert};

#[derive(Deserialize, Debug)]
struct Input {
    date: Option<String>,
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
    // lambda_runtime::run(handler_fn(handler)).await?;
    info!(
        "{}",
        json!(
            handler(
                Input {
                    // date: Some("2000-07-15".to_string()),
                    date: None,
                },
                Context::default(),
            )
            .await?
        )
    );
    Ok(())
}

async fn handler(input: Input, _: Context) -> Result<Output, Error> {
    debug!("Got input: {:?}", input);

    let comic = Dilbert::default().scrape_comic(input.date).await?;

    Ok(Output {
        comic,
        upload_url: "".to_string(),
    })
}
