package main

import (
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gorilla/feeds"
)

func generateFeed(w io.Writer, startDate time.Time, feedLength int, baseURL string) error {
	feed := &feeds.Feed{
		Title:       "Dilbert",
		Link:        &feeds.Link{Href: "http://dilbert.com"},
		Description: "Dilbert Daily Strip",
	}

	for i := 0; i < feedLength; i++ {
		day := startDate.AddDate(0, 0, -i).Truncate(24 * time.Hour)
		date := fmt.Sprintf("%d-%02d-%02d", day.Year(), day.Month(), day.Day())
		url := fmt.Sprintf("%s%s.gif", baseURL, date)

		feed.Add(&feeds.Item{
			Title:       fmt.Sprintf("Dilbert - %s", date),
			Link:        &feeds.Link{Href: url},
			Description: fmt.Sprintf(`<img src="%s">`, url),
			Id:          url,
			Created:     day,
		})
	}

	return feed.WriteRss(w)
}

func uploadFeed(r io.Reader, bucketName, feedPath string) (string, error) {
	sess, err := session.NewSession()
	if err != nil {
		return "", err
	}

	upload, err := s3manager.NewUploader(sess).Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(feedPath),
		Body:        r,
		ContentType: aws.String("text/xml; charset=utf-8"),
	})
	if err != nil {
		return "", err
	}

	return upload.Location, nil
}
