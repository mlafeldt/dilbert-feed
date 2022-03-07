use anyhow::{anyhow, Context, Result};
use aws_sdk_s3::Client;
use chrono::{DateTime, Duration, NaiveDate, Utc};
use derive_builder::Builder;
use futures::future;
use rss::{ChannelBuilder, GuidBuilder, Item, ItemBuilder};
use url::Url;

#[derive(Builder, Debug)]
#[builder(setter(into))]
pub struct Feed {
    bucket_name: String,
    strips_dir: String,
    start_date: NaiveDate,
    #[builder(default = "30")]
    feed_length: i32,
    #[builder(default)]
    s3_client: Option<Client>,
}

impl Feed {
    pub async fn xml(&self) -> Result<String> {
        let items = future::try_join_all(
            (0..self.feed_length)
                .map(|i| self.start_date - Duration::days(i.into()))
                .map(|date| async move {
                    let url: Url = format!(
                        "https://{}.s3.amazonaws.com/{}/{}.gif",
                        self.bucket_name, self.strips_dir, date
                    )
                    .parse()?;
                    let item = ItemBuilder::default()
                        .title(self.title(date).await?)
                        .link(url.to_string())
                        .description(format!(r#"<img src="{}">"#, url))
                        .guid(GuidBuilder::default().value(url).build())
                        .pub_date(DateTime::<Utc>::from_utc(date.and_hms(0, 0, 0), Utc).to_rfc2822())
                        .build();
                    Ok(item) as Result<Item>
                }),
        )
        .await?;

        let channel = ChannelBuilder::default()
            .title("Dilbert")
            .link("https://dilbert.com")
            .description("Dilbert Daily Strip")
            .items(items)
            .build();

        let buf = channel.pretty_write_to(Vec::new(), b' ', 2)?;

        Ok(String::from_utf8(buf)?)
    }

    async fn title(&self, date: NaiveDate) -> Result<String> {
        match &self.s3_client {
            Some(client) => {
                let key = format!("{}/{}.gif", self.strips_dir, date);
                let metadata = client
                    .head_object()
                    .bucket(&self.bucket_name)
                    .key(&key)
                    .send()
                    .await
                    .with_context(|| format!("failed to get metadata from object {}", &key))?
                    .metadata
                    .ok_or_else(|| anyhow!("metadata not found"))?;

                let title = metadata
                    .get("title")
                    .ok_or_else(|| anyhow!("title not found in metadata"))?;

                Ok(title.into())
            }
            None => Ok(format!("Dilbert - {}", date)),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::NaiveDate;
    use pretty_assertions::assert_eq;

    // TODO: figure out how to stub/mock S3 client
    #[tokio::test]
    async fn test_xml() {
        let feed = FeedBuilder::default()
            .bucket_name("dilbert-feed-test")
            .strips_dir("strips")
            .start_date(NaiveDate::from_ymd(2018, 10, 1))
            .feed_length(3)
            .build()
            .unwrap();
        let got = feed.xml().await.unwrap();

        let want = include_str!("testdata/feed.xml").trim();

        assert_eq!(got, want);
    }
}
