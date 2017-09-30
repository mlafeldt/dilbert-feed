package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Comic struct {
	Date  string
	Title string
	Image string
	Strip string
}

func comicForDate(date string) (*Comic, error) {
	strip := "http://dilbert.com/strip/" + date

	doc, err := goquery.NewDocument(strip)
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

	return &Comic{
		Date:  date,
		Title: title,
		Image: image,
		Strip: strip,
	}, nil
}

const atomTemplate = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
 <title>Dilbert</title>
 {{ range . }}
 <entry>
   <title>{{ .Title }}</title>
   <link href="{{ .Strip }}"/>
   <updated>{{ .Date }}</updated>
   <content type="html"><p><img src="{{ .Image }}" title="{{ .Title }}"></p></content>
 </entry>
 {{ end }}
</feed>
`

func main() {
	var comics []Comic
	now := time.Now()

	for i := 1; i <= 30; i++ {
		t := now.AddDate(0, 0, -i)
		date := fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), t.Day())
		comic, err := comicForDate(date)
		if err != nil {
			log.Fatal(err)
		}
		comics = append(comics, *comic)
	}

	t, err := template.New("feed").Parse(atomTemplate)
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(os.Stdout, comics)
}
