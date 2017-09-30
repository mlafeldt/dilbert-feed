package main

import (
	"fmt"
	"log"
	"os"

	"github.com/PuerkitoBio/goquery"
)

type comic struct {
	title string
	image string
}

func comicForDate(date string) (*comic, error) {
	doc, err := goquery.NewDocument("http://dilbert.com/strip/" + date)
	if err != nil {
		return nil, err
	}

	var title, image string

	doc.Find(".img-comic-container").Each(func(i int, s *goquery.Selection) {
		img := s.Find("img")
		if v, ok := img.Attr("alt"); ok {
			title = v
		}
		if v, ok := img.Attr("src"); ok {
			image = v
		}
	})

	if title == "" {
		return nil, fmt.Errorf("title not found")
	}
	if image == "" {
		return nil, fmt.Errorf("image not found")
	}

	return &comic{title, image}, nil
}

func main() {
	for _, date := range os.Args[1:] {
		comic, err := comicForDate(date)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s %s\n", comic.image, comic.title)
	}
}
