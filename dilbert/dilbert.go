package dilbert

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Comic struct {
	Date     string
	Title    string
	ImageURL string
	StripURL string
}

func ComicForDate(date string) (*Comic, error) {
	stripURL := "http://dilbert.com/strip/" + date

	req, err := http.NewRequest("GET", stripURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, err
	}

	var title, imageURL string

	doc.Find(".img-comic-container").Each(func(i int, s *goquery.Selection) {
		img := s.Find("img")
		if v, ok := img.Attr("alt"); ok {
			title = v
		}
		if v, ok := img.Attr("src"); ok {
			imageURL = v
		}
	})

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

func (comic *Comic) DownloadImage(filepath string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	req, err := http.NewRequest("GET", comic.ImageURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
