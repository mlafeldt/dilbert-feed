package main

import (
	"log"
	"os"

	"github.com/mlafeldt/dilbert-feed/dilbert"
)

func main() {
	for _, date := range os.Args[1:] {
		comic, err := dilbert.ComicForDate(date)
		if err != nil {
			log.Fatal(err)
		}

		filepath := date + ".gif"

		log.Printf("Downloading strip %s to %s\n", comic.StripURL, filepath)

		if err := comic.DownloadImage(filepath); err != nil {
			log.Fatal(err)
		}
	}
}
