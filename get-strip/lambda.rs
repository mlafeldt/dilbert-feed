use chrono::Datelike;
use lambda_runtime::{handler_fn, Context, Error};
use log::{debug, info};
use select::document::Document;
use select::predicate::Class;
use serde::{Deserialize, Serialize};
use serde_json::json;
// use std::env;

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
                    date: Some("2000-07-15".to_string()),
                    // date: None,
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

#[derive(Serialize, Deserialize, PartialEq, Debug)]
struct Comic {
    date: String,
    title: String,
    image_url: String,
    strip_url: String,
}

struct Dilbert {
    base_url: String,
}

impl Default for Dilbert {
    fn default() -> Self {
        Dilbert {
            base_url: "https://dilbert.com".to_string(),
        }
    }
}

impl Dilbert {
    pub fn new(base_url: String) -> Self {
        Self { base_url }
    }

    pub async fn scrape_comic(self, date: Option<String>) -> Result<Comic, Error> {
        let date = date.unwrap_or_else(|| {
            let now = chrono::Utc::now();
            format!("{}-{:02}-{:02}", now.year(), now.month(), now.day())
        });

        let strip_url = format!("{}/strip/{}", self.base_url, date);
        let resp = reqwest::get(&strip_url).await?.error_for_status()?;
        let body = resp.text().await?;
        let document = Document::from(body.as_ref());
        let container = document.find(Class("comic-item-container")).next().unwrap();

        // info!("{}", container.attr("data-id").unwrap());
        let title = container.attr("data-title").unwrap();
        let image_url = container.attr("data-image").unwrap();

        Ok(Comic {
            date,
            title: title.to_string(),
            image_url: image_url.to_string(),
            strip_url,
        })
    }
}
