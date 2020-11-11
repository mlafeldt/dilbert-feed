package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/google/go-cmp/cmp"
)

type mockS3Client struct {
	s3iface.S3API
	Titles map[string]*string
}

func (m *mockS3Client) HeadObjectWithContext(ctx context.Context, input *s3.HeadObjectInput, opt ...request.Option) (*s3.HeadObjectOutput, error) {
	return &s3.HeadObjectOutput{
		Metadata: map[string]*string{
			"Title": m.Titles[aws.StringValue(input.Key)],
		},
	}, nil
}

func TestFeedGenerator(t *testing.T) {
	var buf bytes.Buffer
	date, _ := time.Parse("2006-01-02", "2018-10-01")

	mockClient := &mockS3Client{
		Titles: map[string]*string{
			"strips/2018-10-01.gif": aws.String("Use Company Products"),
			"strips/2018-09-29.gif": aws.String("Fine Lines"),
		},
	}

	g := FeedGenerator{
		BucketName: "dilbert-feed-test",
		StripsDir:  "strips",
		StartDate:  date,
		FeedLength: 3,
		S3Client:   mockClient,
	}

	if err := g.Generate(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}

	got, err := ioutil.ReadAll(&buf)
	if err != nil {
		t.Fatal(err)
	}

	want, err := ioutil.ReadFile("testdata/feed.xml")
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(strings.TrimSpace(string(want)), string(got)); diff != "" {
		t.Error(diff)
	}
}

type mockS3Uploader struct {
	s3manageriface.UploaderAPI
}

func (m *mockS3Uploader) UploadWithContext(ctx context.Context, input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	return &s3manager.UploadOutput{
		Location: fmt.Sprintf("https://%s.s3.amazonaws.com/%s",
			aws.StringValue(input.Bucket), aws.StringValue(input.Key)),
	}, nil
}

func TestFeedUploader(t *testing.T) {
	u := FeedUploader{
		BucketName: "dilbert-feed-test",
		FeedPath:   "some/path/feed.xml",
		S3Uploader: &mockS3Uploader{},
	}
	want := "https://dilbert-feed-test.s3.amazonaws.com/some/path/feed.xml"

	got, err := u.Upload(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Error(diff)
	}
}
