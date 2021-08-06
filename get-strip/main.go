package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/kelseyhightower/envconfig"

	"github.com/mlafeldt/dilbert-feed/dilbert"
)

// Input is the input passed to the Lambda function.
type Input struct {
	Date string `json:"date"`
}

// Output is the output returned by the Lambda function.
type Output struct {
	*dilbert.Comic
	UploadURL string `json:"upload_url"`
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, input Input) (*Output, error) {
	var env struct {
		BucketName string `envconfig:"BUCKET_NAME" required:"true"`
		StripsDir  string `envconfig:"STRIPS_DIR" required:"true"`
	}
	if err := envconfig.Process("", &env); err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] env = %+v", env)

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

	comic, err := dilbert.ScrapeComic(ctx, date)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] comic = %+v", comic)

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Uploading strip %s to bucket %q ...", comic.StripURL, env.BucketName)
	cp := StripCopier{
		BucketName: env.BucketName,
		StripsDir:  env.StripsDir,
		S3Uploader: s3manager.NewUploader(sess),
		HTTPClient: http.DefaultClient,
	}
	stripURL, err := cp.Copy(ctx, comic)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Upload completed: %s", stripURL)
	return &Output{comic, stripURL}, nil
}

// StripCopier can copy a comic strip from dilbert.com to S3.
type StripCopier struct {
	BucketName string
	StripsDir  string
	S3Uploader s3manageriface.UploaderAPI
	HTTPClient *http.Client
}

// Copy copies a comic strip from dilbert.com to S3.
func (cp *StripCopier) Copy(ctx context.Context, comic *dilbert.Comic) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", comic.ImageURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := cp.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %s", resp.Status)
	}

	upload, err := cp.S3Uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket:      aws.String(cp.BucketName),
		Key:         aws.String(fmt.Sprintf("%s/%s.gif", cp.StripsDir, comic.Date)),
		ContentType: aws.String("image/gif"),
		// Add strip title to metadata for gen-feed to create nicer RSS feed entries
		Metadata: map[string]*string{
			"Title": aws.String(comic.Title),
		},
		Body: resp.Body,
	})
	if err != nil {
		return "", err
	}

	return upload.Location, nil
}
