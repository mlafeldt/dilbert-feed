package main

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/mlafeldt/dilbert-feed/dilbert"
)

func main() {
	lambda.Start(handler)
}

func handler() error {
	now := time.Now()
	date := fmt.Sprintf("%d-%02d-%02d", now.Year(), now.Month(), now.Day())
	filename := fmt.Sprintf("/tmp/%s.gif", date)

	comic, err := dilbert.ComicForDate(date)
	if err != nil {
		return err
	}

	log.Printf("Downloading strip %s to %s\n", comic.StripURL, filename)

	return comic.DownloadImage(filename)
}
