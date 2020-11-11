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

var baseURL = "https://dilbert.com"

// SetBaseURL overrides the base URL for testing.
func SetBaseURL(url string) {
	baseURL = url
}

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var title, imageURL string

	if container := doc.Find(".img-comic-container"); container != nil {
		img := container.Find("img")
		if v, ok := img.Attr("alt"); ok {
			title = strings.TrimSpace(strings.TrimSuffix(v, titleSuffix))
		}
		if v, ok := img.Attr("src"); ok {
			imageURL = strings.TrimSpace(v)
		}
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
