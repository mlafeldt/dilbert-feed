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
	UploadURL string `json:"upload_url"`
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

	comic, err := dilbert.NewComic(date)
	if err != nil {
		return nil, err
	}

	log.Printf("DEBUG: %+v", comic)

	bucket := os.Getenv("BUCKET_NAME")

	log.Printf("INFO: Uploading strip %q to bucket %q ...", comic.StripURL, bucket)

	client := &http.Client{Timeout: 5 * time.Second}
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

	table := os.Getenv("DYNAMODB_TABLE")

	log.Printf("INFO: Writing metadata to DynamoDB table %q ...", table)

	result := &Comic{comic, uploadResult.Location}
	av, err := dynamodbattribute.MarshalMap(result)
	if err != nil {
		return nil, err
	}

	_, err = dynamodb.New(sess).PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      av,
	})
	if err != nil {
		return nil, err
	}

	log.Printf("DEBUG: %+v", result)
	log.Print("INFO: Done!")

	return result, nil
}
