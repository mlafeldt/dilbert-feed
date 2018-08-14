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
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/mlafeldt/dilbert-feed/dilbert"
)

type Input struct {
	Date string `json:"date"`
}

type Comic struct {
	*dilbert.Comic
	BucketPath string `json:"bucket_path"`
}

func main() {
	lambda.Start(handler)
}

func handler(input Input) (*Comic, error) {
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

	comic, err := dilbert.ComicForDate(date)
	if err != nil {
		return nil, err
	}

	log.Printf("DEBUG: %+v", comic)

	bucket := os.Getenv("BUCKET_NAME")
	path := fmt.Sprintf("strips/%s/%s/%s.gif", year, month, date)

	log.Printf("INFO: Copying strip %s to s3://%s/%s ...", comic.StripURL, bucket, path)

	req, err := http.NewRequest("GET", comic.ImageURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	svc := s3manager.NewUploader(sess)
	_, err = svc.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(path),
		Body:        resp.Body,
		ContentType: aws.String("image/gif"),
	})
	if err != nil {
		return nil, err
	}

	table := os.Getenv("DYNAMODB_TABLE")

	result := &Comic{comic, path}

	log.Printf("INFO: Writing data to DynamoDB table %q...", table)

	av, err := dynamodbattribute.MarshalMap(result)
	if err != nil {
		return nil, err
	}

	dynamo := dynamodb.New(sess)
	_, err = dynamo.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      av,
	})
	if err != nil {
		return nil, err
	}

	log.Print("INFO: Done!")

	return result, nil
}
