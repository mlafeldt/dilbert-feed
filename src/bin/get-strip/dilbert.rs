use anyhow::{anyhow, bail, Result};
use chrono::{NaiveDate, Utc};
use derive_builder::Builder;
use select::document::Document;
use select::predicate::Class;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, PartialEq, Debug)]
pub struct Comic {
    pub date: NaiveDate,
    pub title: String,
    pub image_url: String,
    pub strip_url: String,
}

#[derive(Builder, Debug)]
pub struct Client {
    #[builder(default = "String::from(\"https://dilbert.com\")")]
    base_url: String,
    #[builder(default)]
    http_client: reqwest::Client,
}

impl Default for Client {
    fn default() -> Self {
        ClientBuilder::default().build().unwrap()
    }
}

impl Client {
    pub async fn scrape_comic(&self, date: Option<NaiveDate>) -> Result<Comic> {
        let date = date.unwrap_or_else(|| Utc::today().naive_utc());
        let strip_url = self.strip_url(date);
        let body = self
            .http_client
            .get(&strip_url)
            .send()
            .await?
            .error_for_status()?
            .text()
            .await?;

        let document = Document::from(body.as_ref());
        let container = document
            .find(Class("comic-item-container"))
            .next()
            .ok_or_else(|| anyhow!("comic metadata not found"))?;

        if container.attr("data-id").unwrap_or_default() != date.to_string() {
            bail!("no comic found for date {}", date);
        }

        let title = container
            .attr("data-title")
            .ok_or_else(|| anyhow!("title not found"))?
            .trim()
            .to_string();
        let image_url = container
            .attr("data-image")
            .ok_or_else(|| anyhow!("image URL not found"))?
            .trim()
            .to_string();

        Ok(Comic {
            date,
            title,
            image_url,
            strip_url,
        })
    }

    fn strip_url(&self, date: NaiveDate) -> String {
        format!("{}/strip/{}", self.base_url, date)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use pretty_assertions::assert_eq;
    use wiremock::matchers::{method, path};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    struct Test {
        comic: Comic,
        html: &'static str,
    }

    fn tests(base_url: &str) -> Vec<Test> {
        vec![
            Test {
                comic: Comic {
                    date: NaiveDate::from_ymd(2000, 1, 1),
                    title: "Dilbert Comic for 2000-01-01".to_string(),
                    image_url: "https://assets.amuniversal.com/bdc8a4d06d6401301d80001dd8b71c47".to_string(),
                    strip_url: format!("{}/strip/2000-01-01", base_url),
                },
                html: include_str!("testdata/strip/2000-01-01"),
            },
            Test {
                comic: Comic {
                    date: NaiveDate::from_ymd(2018, 10, 30),
                    title: "Intentionally Underbidding".to_string(),
                    image_url: "https://assets.amuniversal.com/cda546d0a88c01365b26005056a9545d".to_string(),
                    strip_url: format!("{}/strip/2018-10-30", base_url),
                },
                html: include_str!("testdata/strip/2018-10-30"),
            },
            Test {
                comic: Comic {
                    date: NaiveDate::from_ymd(2019, 11, 2),
                    title: "Multiple Choice".to_string(),
                    image_url: "https://assets.amuniversal.com/ce7ec130d6480137c832005056a9545d".to_string(),
                    strip_url: format!("{}/strip/2019-11-02", base_url),
                },
                html: include_str!("testdata/strip/2019-11-02"),
            },
            Test {
                comic: Comic {
                    date: NaiveDate::from_ymd(2020, 11, 11),
                    title: "Elbonian Words".to_string(),
                    image_url: "https://assets.amuniversal.com/f25312c0fb5b01382ef9005056a9545d".to_string(),
                    strip_url: format!("{}/strip/2020-11-11", base_url),
                },
                html: include_str!("testdata/strip/2020-11-11"),
            },
        ]
    }

    #[tokio::test]
    async fn test_scrape_comic() {
        let server = MockServer::start().await;

        for t in tests(&server.uri()).iter() {
            let resp = ResponseTemplate::new(200).set_body_raw(t.html, "text/html");

            Mock::given(method("GET"))
                .and(path(format!("/strip/{}", t.comic.date)))
                .respond_with(resp)
                .expect(1)
                .mount(&server)
                .await;

            let comic = ClientBuilder::default()
                .base_url(server.uri())
                .build()
                .unwrap()
                .scrape_comic(Some(t.comic.date))
                .await
                .unwrap();

            assert_eq!(comic, t.comic);

            server.reset().await;
        }
    }
}
