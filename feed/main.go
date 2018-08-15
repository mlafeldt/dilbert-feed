package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/mlafeldt/dilbert-feed/dilbert"
)

const feedLength = 30

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

type Input struct {
	Date string `json:"date"`
}

type Output struct {
	*dilbert.Comic
	UploadURL string `json:"upload_url"`
}

func main() {
	lambda.Start(handler)
}

func handler(input Input) (*Output, error) {
	var date string

	if input.Date != "" {
		date = strings.TrimSpace(input.Date)
		if len(date) != 10 {
			return nil, fmt.Errorf("input date %q has invalid length", date)
		}
		if len(strings.Split(date, "-")) != 3 {
			return nil, fmt.Errorf("input date %q has invalid format", date)
		}
	}

	comic, err := dilbert.NewComic(date)
	if err != nil {
		return nil, err
	}

	log.Printf("DEBUG: %+v", comic)

	bucket := os.Getenv("BUCKET_NAME")

	log.Printf("INFO: Uploading strip %s to bucket %q ...", comic.StripURL, bucket)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(comic.ImageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	uploadResult, err := s3manager.NewUploader(sess).Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(fmt.Sprintf("strips/%s.gif", comic.Date)),
		Body:        resp.Body,
		ContentType: aws.String("image/gif"),
	})
	if err != nil {
		return nil, err
	}

	output := Output{comic, uploadResult.Location}

	log.Printf("INFO: Upload completed: %s", output.UploadURL)

	// TODO: move the following to a separate Lambda function

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
		t := now.AddDate(0, 0, -i)
		date := fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), t.Day())
		comics = append(comics, dilbert.Comic{
			Date:     date,
			ImageURL: fmt.Sprintf("https://%s.s3.%s.amazonaws.com/strips/%s.gif", bucket, bucketRegion, date),
		})
	}

	templ, err := template.New("feed").Parse(feedTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	templ.Execute(&buf, comics)

	uploadResult, err = s3manager.NewUploader(sess).Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String("v0/rss.xml"),
		Body:        &buf,
		ContentType: aws.String("text/xml; charset=utf-8"),
	})
	if err != nil {
		return nil, err
	}

	log.Printf("INFO: Upload completed: %s", uploadResult.Location)

	return &output, nil
}
