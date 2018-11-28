package main

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/epsagon/epsagon-go/epsagon"
	"github.com/kelseyhightower/envconfig"
)

const (
	defaultFeedPath   = "v0/rss.xml"
	defaultFeedLength = 30
)

// Input is the input passed to the Lambda function.
type Input struct{}

// Output is the output returned by the Lambda function.
type Output struct {
	FeedURL string `json:"feed_url"`
}

func main() {
	lambda.Start(epsagon.WrapLambdaHandler(
		&epsagon.Config{ApplicationName: "dilbert-feed"}, handler))
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
	log.Printf("[DEBUG] env = %+v", env)

	now := time.Now()
	baseURL := fmt.Sprintf("https://%s/%s", env.DomainName, env.BucketPrefix)
	var buf bytes.Buffer

	log.Printf("[INFO] Generating feed for date %s ...", now.Format(time.RFC3339))
	if err := generateFeed(&buf, now, defaultFeedLength, baseURL); err != nil {
		return nil, err
	}

	log.Printf("[INFO] Uploading feed to bucket %q with path %q ...", env.BucketName, defaultFeedPath)
	feedURL, err := uploadFeed(&buf, env.BucketName, defaultFeedPath)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Upload completed: %s", feedURL)
	return &Output{feedURL}, nil
}
