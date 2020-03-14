package main

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/kelseyhightower/envconfig"
)

const feedLength = 30

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
		BucketName string `envconfig:"BUCKET_NAME" required:"true"`
		StripsDir  string `envconfig:"STRIPS_DIR" required:"true"`
		FeedPath   string `envconfig:"FEED_PATH" required:"true"`
	}
	if err := envconfig.Process("", &env); err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] env = %+v", env)

	var (
		now     = time.Now()
		baseURL = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", env.BucketName, env.StripsDir)
		buf     bytes.Buffer
	)

	log.Printf("[INFO] Generating feed for date %s ...", now.Format(time.RFC3339))
	if err := generateFeed(&buf, now, feedLength, baseURL); err != nil {
		return nil, err
	}

	log.Printf("[INFO] Uploading feed to bucket %q with path %q ...", env.BucketName, env.FeedPath)
	feedURL, err := uploadFeed(&buf, env.BucketName, env.FeedPath)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Upload completed: %s", feedURL)
	return &Output{feedURL}, nil
}
