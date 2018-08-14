package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
	ImageURL string `json:"image_url"`
}

func main() {
	lambda.Start(handler)
}

func handler(input Input) (*Output, error) {
	now := time.Now()
	year := strconv.Itoa(now.Year())
	month := fmt.Sprintf("%02d", now.Month())
	day := fmt.Sprintf("%02d", now.Day())
	date := strings.Join([]string{year, month, day}, "-")

	if input.Date != "" {
		date = strings.TrimSpace(input.Date)
		if len(date) != 10 {
			return nil, fmt.Errorf("input date %q has invalid length", date)
		}
		parts := strings.SplitN(date, "-", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("input date %q has invalid format", date)
		}
		year, month, day = parts[0], parts[1], parts[2]
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
		Key:         aws.String(fmt.Sprintf("strips/%s/%s/%s.gif", year, month, date)),
		Body:        resp.Body,
		ContentType: aws.String("image/gif"),
	})
	if err != nil {
		return nil, err
	}

	output := Output{ImageURL: uploadResult.Location}

	log.Printf("INFO: Upload completed: %s", output.ImageURL)

	return &output, nil
}
