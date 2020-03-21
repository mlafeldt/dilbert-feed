package main

import (
	"bytes"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/kelseyhightower/envconfig"
)

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
		now = time.Now()
		buf bytes.Buffer
	)

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Generating feed for date %s ...", now.Format(time.RFC3339))
	g := FeedGenerator{
		BucketName: env.BucketName,
		StripsDir:  env.StripsDir,
		StartDate:  now,
		FeedLength: 30,
		S3Client:   s3.New(sess),
	}
	if err := g.Generate(&buf); err != nil {
		return nil, err
	}

	log.Printf("[INFO] Uploading feed to bucket %q with path %q ...", env.BucketName, env.FeedPath)
	u := FeedUploader{
		BucketName: env.BucketName,
		FeedPath:   env.FeedPath,
		S3Uploader: s3manager.NewUploader(sess),
	}
	feedURL, err := u.Upload(&buf)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Upload completed: %s", feedURL)
	return &Output{feedURL}, nil
}
