# dilbert-feed

Enjoy [Dilbert](https://dilbert.com/) in your RSS feed reader without any ads!

Unfortunetly, Dilbert's official feed now forces you to go to the website:

> Dilbert readers - Please visit Dilbert.com to read this feature. Due to changes with our feeds, we are now making this RSS feed a link to Dilbert.com.

This serverless application provides a custom feed with direct access to Dilbert comics that gets updated every day.

For some background information, check out my article on [Recreational Programming with Serverless](https://sharpend.io/recreational-programming-with-serverless/).

## Architecture

![](architecture.png)

## Deployment

Follow these steps to deploy your own dilbert-feed instance to AWS.

Set AWS region and credentials in environment:

```console
export AWS_REGION=eu-central-1
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
```

Configure the heartbeat URL, e.g. as provided by [Healthchecks.io](https://healthchecks.io/):

```console
aws ssm put-parameter --overwrite --name /dilbert-feed-dev/heartbeat-endpoint --type String --value <url>
aws ssm put-parameter --overwrite --name /dilbert-feed-prod/heartbeat-endpoint --type String --value <url>
```

Deploy the stack (requires Node.js and Go to be installed):

```console
# Bootstrap AWS CDK once
make bootstrap

# Deploy development environment
make dev

# Deploy production environment
make prod
```

Among other things, the stack outputs will show the URL of the RSS feed, which you can then subscribe:

```console
dilbert-feed-prod.FeedUrl = https://dilbert-feed-example.s3.eu-central-1.amazonaws.com/v1/rss.xml
```

## Usage

The serverless stack will update the feed automatically. However, you can also invoke the Lambda functions manually.

Get the comic strip for today:

```console
$ ./invoke dilbert-feed-prod-get-strip
{
  "date": "2019-10-22",
  "title": "Best Employees",
  "image_url": "https://assets.amuniversal.com/87b83e10c7460137c2df005056a9545d",
  "strip_url": "https://dilbert.com/strip/2019-10-22",
  "upload_url": "https://dilbert-feed-example.s3.eu-central-1.amazonaws.com/strips/2019-10-22.gif"
}
```

Get the comic strip for a specific date:

```console
$ ./invoke dilbert-feed-prod-get-strip --payload '{"date":"2016-01-01"}'
{
  "date": "2016-01-01",
  "title": "Forgetting Meetings",
  "image_url": "https://assets.amuniversal.com/1a6be66079e101332131005056a9545d",
  "strip_url": "https://dilbert.com/strip/2016-01-01",
  "upload_url": "https://dilbert-feed-example.s3.eu-central-1.amazonaws.com/strips/2016-01-01.gif"
}
```

Get the comic strips for the last 30 days:

```console
for i in $(seq 0 30); do date=$(gdate -I -d "today -$i days"); ./invoke dilbert-feed-prod-get-strip --payload "{\"date\":\"$date\"}"; done
```

Generate the RSS feed:

```console
$ ./invoke dilbert-feed-prod-gen-feed
{
  "feed_url": "https://dilbert-feed-example.s3.eu-central-1.amazonaws.com/v1/rss.xml"
}
```
