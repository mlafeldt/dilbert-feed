use chrono::Datelike;
use lambda_runtime::Error;
use regex::Regex;
use select::document::Document;
use select::predicate::Class;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, PartialEq, Debug)]
pub struct Comic {
    pub date: String,
    pub title: String,
    pub image_url: String,
    pub strip_url: String,
}

#[derive(Debug)]
pub struct Dilbert {
    base_url: String,
}

impl Default for Dilbert {
    fn default() -> Self {
        Self::new("https://dilbert.com")
    }
}

impl Dilbert {
    pub fn new(base_url: &str) -> Self {
        Self {
            base_url: base_url.to_string(),
        }
    }

    pub async fn scrape_comic(self, date: Option<String>) -> Result<Comic, Error> {
        let date = Self::date_or_today(date)?;
        let strip_url = self.strip_url(&date);
        let resp = reqwest::get(&strip_url).await?.error_for_status()?;
        let body = resp.text().await?;

        let document = Document::from(body.as_ref());
        let container = document
            .find(Class("comic-item-container"))
            .next()
            .ok_or("comic metadata not found")?;

        if date != container.attr("data-id").unwrap_or_default() {
            return Err("comic not found for date".into());
        }

        let title = container
            .attr("data-title")
            .ok_or("title not found")?
            .trim()
            .to_string();
        let image_url = container
            .attr("data-image")
            .ok_or("image URL not found")?
            .trim()
            .to_string();

        Ok(Comic {
            date,
            title,
            image_url,
            strip_url,
        })
    }

    fn date_or_today(date: Option<String>) -> Result<String, Error> {
        match date {
            Some(date) => {
                let re = Regex::new(r"^\d{4}-\d{2}-\d{2}$").unwrap();
                if !re.is_match(&date) {
                    return Err(format!("invalid date format: {}", date).into());
                }
                Ok(date)
            }
            None => {
                let now = chrono::Utc::now();
                Ok(format!("{}-{:02}-{:02}", now.year(), now.month(), now.day()))
            }
        }
    }

    fn strip_url(self, date: &str) -> String {
        format!("{}/strip/{}", self.base_url, date)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
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
                    date: "2000-01-01".to_string(),
                    title: "Dilbert Comic for 2000-01-01".to_string(),
                    image_url: "https://assets.amuniversal.com/bdc8a4d06d6401301d80001dd8b71c47".to_string(),
                    strip_url: format!("{}/strip/2000-01-01", base_url),
                },
                html: include_str!("testdata/strip/2000-01-01"),
            },
            Test {
                comic: Comic {
                    date: "2018-10-30".to_string(),
                    title: "Intentionally Underbidding".to_string(),
                    image_url: "https://assets.amuniversal.com/cda546d0a88c01365b26005056a9545d".to_string(),
                    strip_url: format!("{}/strip/2018-10-30", base_url),
                },
                html: include_str!("testdata/strip/2018-10-30"),
            },
            Test {
                comic: Comic {
                    date: "2019-11-02".to_string(),
                    title: "Multiple Choice".to_string(),
                    image_url: "https://assets.amuniversal.com/ce7ec130d6480137c832005056a9545d".to_string(),
                    strip_url: format!("{}/strip/2019-11-02", base_url),
                },
                html: include_str!("testdata/strip/2019-11-02"),
            },
            Test {
                comic: Comic {
                    date: "2020-11-11".to_string(),
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

            let comic = Dilbert::new(&server.uri())
                .scrape_comic(Some(t.comic.date.to_owned()))
                .await
                .unwrap();

            assert_eq!(comic, t.comic);

            server.reset().await;
        }
    }
}
