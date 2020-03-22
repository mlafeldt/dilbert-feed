package main

import (
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/gorilla/feeds"
)

// FeedGenerator can build an RSS feed for Dilbert.
type FeedGenerator struct {
	BucketName string
	StripsDir  string
	StartDate  time.Time
	FeedLength int
	S3Client   s3iface.S3API
}

// Generate builds an RSS feed for Dilbert.
func (g *FeedGenerator) Generate(w io.Writer) error {
	feed := &feeds.Feed{
		Title:       "Dilbert",
		Link:        &feeds.Link{Href: "http://dilbert.com"},
		Description: "Dilbert Daily Strip",
	}

	for i := 0; i < g.FeedLength; i++ {
		var (
			day   = g.StartDate.AddDate(0, 0, -i).Truncate(24 * time.Hour)
			date  = fmt.Sprintf("%d-%02d-%02d", day.Year(), day.Month(), day.Day())
			url   = fmt.Sprintf("https://%s.s3.amazonaws.com/%s%s.gif", g.BucketName, g.StripsDir, date)
			title = g.title(date)
		)

		feed.Add(&feeds.Item{
			Title:       title,
			Link:        &feeds.Link{Href: url},
			Description: fmt.Sprintf(`<img src="%s">`, url),
			Id:          url,
			Created:     day,
		})
	}

	return feed.WriteRss(w)
}

func (g *FeedGenerator) title(date string) string {
	title := fmt.Sprintf("Dilbert - %s", date)

	if g.S3Client != nil {
		out, err := g.S3Client.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(g.BucketName),
			Key:    aws.String(fmt.Sprintf("%s%s.gif", g.StripsDir, date)),
		})
		// Silently return fabricated title on error
		if err == nil {
			if v := aws.StringValue(out.Metadata["Title"]); v != "" {
				title = v
			}
		}
	}

	return title
}

// FeedUploader can upload a Dilbert feed to S3.
type FeedUploader struct {
	BucketName string
	FeedPath   string
	S3Uploader s3manageriface.UploaderAPI
}

// Upload uploads a Dilbert feed to S3.
func (u *FeedUploader) Upload(r io.Reader) (string, error) {
	upload, err := u.S3Uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(u.BucketName),
		Key:         aws.String(u.FeedPath),
		Body:        r,
		ContentType: aws.String("text/xml; charset=utf-8"),
	})
	if err != nil {
		return "", err
	}

	return upload.Location, nil
}
