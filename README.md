# dilbert-feed

Enjoy the [Dilbert Feed](http://dilbert.com/) Without Ads!

## Usage

Get the comic strip for today:

```console
$ sls invoke -f get-strip
{
    "date": "2018-08-15",
    "title": "How Dilbert Can Help",
    "image_url": "http://assets.amuniversal.com/b813c1e0718901364736005056a9545d",
    "strip_url": "http://dilbert.com/strip/2018-08-15",
    "upload_url": "https://dilbert-feed.s3.eu-central-1.amazonaws.com/strips/95a5471ee9873fe1e63ca0617f2b1455c6b0ad7aac2411c044adbe790b33f3e1"
}
```

Get the comic strip for a specific date:

```console
$ sls invoke -f get-strip -d '{"date":"2016-01-01"}'
{
    "date": "2016-01-01",
    "title": "Forgetting Meetings",
    "image_url": "http://assets.amuniversal.com/1a6be66079e101332131005056a9545d",
    "strip_url": "http://dilbert.com/strip/2016-01-01",
    "upload_url": "https://dilbert-feed.s3.eu-central-1.amazonaws.com/strips/3bd7cc3fd7ba7402f5bce09458f4d324cb92ff0d038fea727064e61969cc6291"
}
```

Get the comic strips for the last 30 days:

```console
for i in $(seq 0 30); do date=$(gdate -I -d "today -$i days"); printf "{\"date\":\"%s\"}\n" $date | sls invoke -f get-strip; done
```

Generate the RSS feed:

```console
$ sls invoke -f gen-feed
{
    "feed_url": "https://dilbert-feed.s3.eu-central-1.amazonaws.com/v1/rss.xml"
}
```
