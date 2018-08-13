package main

import (
	"io"
	"log"
	"net/http"
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

		if err := downloadFile(filepath, comic.ImageURL); err != nil {
			log.Fatal(err)
		}
	}
}

func downloadFile(filepath string, url string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
