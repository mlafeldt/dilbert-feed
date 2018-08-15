package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/mlafeldt/dilbert-feed/dilbert"
)

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
	prefix := os.Getenv("BUCKET_PREFIX")

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
		Key:         aws.String(fmt.Sprintf("%s/%s.gif", prefix, comic.Date)),
		Body:        resp.Body,
		ContentType: aws.String("image/gif"),
	})
	if err != nil {
		return nil, err
	}

	log.Printf("INFO: Upload completed: %s", uploadResult.Location)

	return &Output{comic, uploadResult.Location}, nil
}
