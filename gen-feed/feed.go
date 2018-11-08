package main

import (
	"fmt"
	"io"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	epsagonaws "github.com/epsagon/epsagon-go/wrappers/aws/aws-sdk-go/aws"
)

const feedTemplate = `<rss version="2.0">
  <channel>
    <title>Dilbert</title>
    <link>http://dilbert.com</link>
    <description>Dilbert Daily Strip</description>
    {{- range . }}
    <item>
      <title>Dilbert - {{ .Date }}</title>
      <link>{{ .ImageURL }}</link>
      <guid>{{ .ImageURL }}</guid>
      <description>
        <![CDATA[
          <img src="{{ .ImageURL }}">
        ]]>
      </description>
    </item>
    {{- end }}
  </channel>
</rss>
`

type feedItem struct {
	Date     string
	ImageURL string
}

func generateFeed(w io.Writer, startDate time.Time, feedLength int, baseURL string) error {
	var items []feedItem
	for i := 0; i < feedLength; i++ {
		day := startDate.AddDate(0, 0, -i)
		date := fmt.Sprintf("%d-%02d-%02d", day.Year(), day.Month(), day.Day())
		url := fmt.Sprintf("%s%s.gif", baseURL, date)
		items = append(items, feedItem{Date: date, ImageURL: url})
	}

	t, err := template.New("feed").Parse(feedTemplate)
	if err != nil {
		return err
	}

	return t.Execute(w, items)
}

func uploadFeed(r io.Reader, bucketName, feedPath string) (string, error) {
	sess, err := session.NewSession()
	if err != nil {
		return "", err
	}
	sess = epsagonaws.WrapSession(sess)

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
