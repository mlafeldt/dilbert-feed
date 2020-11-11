package dilbert

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Comic describes a Dilbert comic strip.
type Comic struct {
	Date     string `json:"date"`
	Title    string `json:"title"`
	ImageURL string `json:"image_url"`
	StripURL string `json:"strip_url"`
}

var baseURL = "https://dilbert.com"

// SetBaseURL overrides the base URL for testing.
func SetBaseURL(url string) {
	baseURL = url
}

// NewComic gets the Dilbert comic strip for the given date.
func NewComic(ctx context.Context, date string) (*Comic, error) {
	if date == "" {
		now := time.Now()
		date = fmt.Sprintf("%d-%02d-%02d", now.Year(), now.Month(), now.Day())
	}

	stripURL := fmt.Sprintf("%s/strip/%s", baseURL, strings.TrimSpace(date))

	req, err := http.NewRequestWithContext(ctx, "GET", stripURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var title, imageURL string

	if container := doc.Find(".comic-item-container"); container != nil {
		if v, ok := container.Attr("data-title"); ok {
			title = strings.TrimSpace(v)
		}
		if v, ok := container.Attr("data-image"); ok {
			imageURL = strings.TrimSpace(v)
		}
	}

	if title == "" {
		return nil, fmt.Errorf("title not found")
	}
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
