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
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mlafeldt/dilbert-feed/dilbert"
)

type comicWithPath struct {
	*dilbert.Comic
	BucketPath string `json:"bucket_path"`
}

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
	path := fmt.Sprintf("strips/%d/%02d/%s.gif", now.Year(), now.Month(), date)

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

	sess, err := session.NewSession()
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}

	svc := s3manager.NewUploader(sess)
	_, err = svc.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(path),
		Body:        resp.Body,
		ContentType: aws.String("image/gif"),
	})
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}

	table := os.Getenv("DYNAMODB_TABLE")

	log.Printf("INFO: Writing comic data to DynamoDB table %q...", table)

	av, err := dynamodbattribute.MarshalMap(comicWithPath{comic, path})
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}

	dynamo := dynamodb.New(sess)
	_, err = dynamo.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      av,
	})
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}

	log.Print("INFO: Done!")
	return nil
}
