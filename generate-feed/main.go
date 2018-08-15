package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"text/template"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/mlafeldt/dilbert-feed/dilbert"
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

type Input struct{}

type Output struct {
	FeedURL string `json:"feed_url"`
}

func main() {
	lambda.Start(handler)
}

func handler(input Input) (*Output, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	bucket := os.Getenv("BUCKET_NAME")
	prefix := os.Getenv("BUCKET_PREFIX")

	bucketLocation, err := s3.New(sess).GetBucketLocation(&s3.GetBucketLocationInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return nil, err
	}
	bucketRegion := aws.StringValue(bucketLocation.LocationConstraint)

	var comics []dilbert.Comic
	now := time.Now()

	for i := 0; i < feedLength; i++ {
		day := now.AddDate(0, 0, -i)
		date := fmt.Sprintf("%d-%02d-%02d", day.Year(), day.Month(), day.Day())
		comics = append(comics, dilbert.Comic{
			Date: date,
			ImageURL: fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s/%s.gif",
				bucket, bucketRegion, prefix, date),
		})
	}

	templ, err := template.New("feed").Parse(feedTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	templ.Execute(&buf, comics)

	uploadResult, err := s3manager.NewUploader(sess).Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(feedPath),
		Body:        &buf,
		ContentType: aws.String("text/xml; charset=utf-8"),
	})
	if err != nil {
		return nil, err
	}

	log.Printf("INFO: Upload completed: %s", uploadResult.Location)

	return &Output{uploadResult.Location}, nil
}
