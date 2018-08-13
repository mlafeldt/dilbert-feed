package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/mlafeldt/dilbert-feed/dilbert"
)

const atomTemplate = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
 <title>Dilbert</title>
 {{ range . }}
 <entry>
   <title>{{ .Title }}</title>
   <link href="{{ .Strip }}"/>
   <updated>{{ .Date }}</updated>
   <content type="html"><p><img src="{{ .Image }}" title="{{ .Title }}"/></p></content>
 </entry>
 {{ end }}
</feed>
`

func main() {
	port := "3000"
	if v := os.Getenv("PORT"); v != "" {
		port = v
	}
	log.Printf("Listening on port %s", port)

	http.HandleFunc("/v1/atom.xml", handler)
	http.ListenAndServe(":"+port, nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	var comics []dilbert.Comic
	now := time.Now()

	for i := 1; i <= 30; i++ {
		t := now.AddDate(0, 0, -i)
		date := fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), t.Day())
		comic, err := dilbert.ComicForDate(date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("%+v\n", comic)
		comics = append(comics, *comic)
	}

	w.Header().Set("Content-Type", "application/xml")

	t, err := template.New("feed").Parse(atomTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	t.Execute(w, comics)
}
