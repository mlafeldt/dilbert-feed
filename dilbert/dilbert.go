package dilbert

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Comic describes a Dilbert comic strip.
type Comic struct {
	Date     string `json:"date"`
	Title    string `json:"title,omitempty"`
	ImageURL string `json:"image_url"`
	StripURL string `json:"strip_url"`
}

const titleSuffix = "- Dilbert by Scott Adams"

var baseURL = "http://dilbert.com"

// NewComic returns the Dilbert comic strip for the given date.
func NewComic(date string) (*Comic, error) {
	if date == "" {
		now := time.Now()
		date = fmt.Sprintf("%d-%02d-%02d", now.Year(), now.Month(), now.Day())
	}

	stripURL := fmt.Sprintf("%s/strip/%s", baseURL, strings.TrimSpace(date))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(stripURL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, err
	}

	var title, imageURL string

	doc.Find(".img-comic-container").Each(func(i int, s *goquery.Selection) {
		img := s.Find("img")
		if v, ok := img.Attr("alt"); ok {
			title = strings.TrimSpace(strings.TrimSuffix(v, titleSuffix))
		}
		if v, ok := img.Attr("src"); ok {
			imageURL = strings.TrimSpace(v)
		}
	})

	if imageURL == "" {
		return nil, fmt.Errorf("image URL not found")
	}

	return &Comic{
		Date:     date,
		Title:    title,
		ImageURL: imageURL,
		StripURL: stripURL,
	}, nil
}
