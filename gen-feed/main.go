package main

import (
	"bytes"
	"fmt"
	"log"
	"text/template"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/kelseyhightower/envconfig"
)

const (
	feedPath   = "v0/rss.xml"
	feedLength = 30
)

const feedTemplate = `<rss version="2.0">
  <channel>
    <title>Dilbert</title>
    <link>http://dilbert.com</link>
    <description>Dilbert Daily Strip</description>
    {{ range . }}
    <item>
      <title>Dilbert - {{ .Date }}</title>
      <link>{{ .ImageURL }}</link>
      <guid>{{ .ImageURL }}</guid>
      <description>
        <![CDATA[
          <img src="{{ .ImageURL }}">
        ]]>
      </description>
    </item>
    {{ end }}
  </channel>
</rss>
`

type feedItem struct {
	Date     string
	ImageURL string
}

// Input is the input passed to the Lambda function.
type Input struct{}

// Output is the output returned by the Lambda function.
type Output struct {
	FeedURL string `json:"feed_url"`
}

func main() {
	lambda.Start(handler)
}

func handler(input Input) (*Output, error) {
	var env struct {
		BucketName   string `envconfig:"BUCKET_NAME" required:"true"`
		BucketPrefix string `envconfig:"BUCKET_PREFIX" required:"true"`
		DomainName   string `envconfig:"DOMAIN_NAME" required:"true"`
	}
	if err := envconfig.Process("", &env); err != nil {
		return nil, err
	}
	log.Printf("DEBUG: env = %+v", env)

	now := time.Now()

	log.Printf("INFO: Generating feed for date %s ...", now.Format(time.RFC3339))

	var items []feedItem
	for i := 0; i < feedLength; i++ {
		day := now.AddDate(0, 0, -i)
		date := fmt.Sprintf("%d-%02d-%02d", day.Year(), day.Month(), day.Day())
		url := fmt.Sprintf("https://%s/%s%s.gif", env.DomainName, env.BucketPrefix, date)
		items = append(items, feedItem{Date: date, ImageURL: url})
	}

	t, err := template.New("feed").Parse(feedTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, items); err != nil {
		return nil, err
	}

	log.Printf("INFO: Uploading feed to bucket %q with path %q ...", env.BucketName, feedPath)

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	upload, err := s3manager.NewUploader(sess).Upload(&s3manager.UploadInput{
		Bucket:      aws.String(env.BucketName),
		Key:         aws.String(feedPath),
		Body:        &buf,
		ContentType: aws.String("text/xml; charset=utf-8"),
	})
	if err != nil {
		return nil, err
	}

	log.Printf("INFO: Upload completed: %s", upload.Location)

	return &Output{upload.Location}, nil
}
