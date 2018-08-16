package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/kelseyhightower/envconfig"

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
	var env struct {
		BucketName   string `envconfig:"BUCKET_NAME" required:"true"`
		BucketPrefix string `envconfig:"BUCKET_PREFIX" required:"true"`
	}
	if err := envconfig.Process("", &env); err != nil {
		return nil, err
	}
	log.Printf("DEBUG: env = %+v", env)

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

	log.Printf("DEBUG: comic = %+v", comic)
	log.Printf("INFO: Uploading strip %s to bucket %q ...", comic.StripURL, env.BucketName)

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

	upload, err := s3manager.NewUploader(sess).Upload(&s3manager.UploadInput{
		Bucket:      aws.String(env.BucketName),
		Key:         aws.String(fmt.Sprintf("%s/%s.gif", env.BucketPrefix, comic.Date)),
		ContentType: aws.String("image/gif"),
		Body:        resp.Body,
	})
	if err != nil {
		return nil, err
	}

	log.Printf("INFO: Upload completed: %s", upload.Location)

	return &Output{comic, upload.Location}, nil
}
