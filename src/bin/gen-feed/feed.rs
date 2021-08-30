use chrono::{DateTime, Duration, NaiveDate, Utc};
use derive_builder::Builder;
use lambda_runtime::Error;
use rss::{ChannelBuilder, GuidBuilder, ItemBuilder};

#[derive(Builder, Debug)]
#[builder(setter(into))]
pub struct Feed {
    bucket_name: String,
    strips_dir: String,
    start_date: NaiveDate,
    #[builder(default = "30")]
    feed_length: i32,
}

impl Feed {
    pub fn xml(&self) -> Result<String, Error> {
        let items: Vec<_> = (0..self.feed_length)
            .map(|i| self.start_date - Duration::days(i.into()))
            .map(|date| {
                let url = format!(
                    "https://{}.s3.amazonaws.com/{}/{}.gif",
                    self.bucket_name, self.strips_dir, date
                );
                ItemBuilder::default()
                    .title(format!("Dilbert - {}", date)) // FIXME
                    .link(url.to_owned())
                    .description(format!(r#"<img src="{}">"#, url))
                    .guid(GuidBuilder::default().value(url).build().unwrap())
                    .pub_date(DateTime::<Utc>::from_utc(date.and_hms(0, 0, 0), Utc).to_rfc2822())
                    .build()
                    .unwrap() // FIXME
            })
            .collect();

        dbg!(&items[0]);

        let channel = ChannelBuilder::default()
            .title("Dilbert")
            .link("https://dilbert.com")
            .description("Dilbert Daily Strip")
            .items(items)
            .build()?;

        let buf = channel.pretty_write_to(Vec::new(), b' ', 2)?;

        Ok(String::from_utf8(buf)?)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::NaiveDate;
    use pretty_assertions::assert_eq;

    #[tokio::test]
    async fn test_xml() {
        let feed = FeedBuilder::default()
            .bucket_name("dilbert-feed-test")
            .strips_dir("strips")
            .start_date(NaiveDate::from_ymd(2018, 10, 1))
            .feed_length(3)
            .build()
            .unwrap();
        let got = feed.xml().unwrap();

        let want = include_str!("testdata/feed.xml").trim();

        assert_eq!(got, want);
    }
}
