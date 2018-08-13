package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mlafeldt/dilbert-feed/dilbert"
)

func main() {
	lambda.Start(handler)
}

func handler() error {
	now := time.Now()
	date := fmt.Sprintf("%d-%02d-%02d", now.Year(), now.Month(), now.Day())

	comic, err := dilbert.ComicForDate(date)
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}

	log.Printf("DEBUG: %+v", comic)

	bucket := os.Getenv("BUCKET_NAME")
	path := fmt.Sprintf("strips/%d/%s.gif", now.Year(), date)

	log.Printf("INFO: Copying strip %s to s3://%s/%s ...", comic.StripURL, bucket, path)

	req, err := http.NewRequest("GET", comic.ImageURL, nil)
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}
	defer resp.Body.Close()

	sess := session.New()
	svc := s3manager.NewUploader(sess)

	_, err = svc.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
		Body:   resp.Body,
	})
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}

	log.Print("INFO: Done!")
	return nil
}
