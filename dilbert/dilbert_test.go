package dilbert_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/mlafeldt/dilbert-feed/dilbert"
)

func TestScrapeComic(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()
	dilbert.SetBaseURL(ts.URL)

	testdata := []*dilbert.Comic{
		{
			Date:     "2000-01-01",
			Title:    "Dilbert Comic for 2000-01-01",
			ImageURL: "https://assets.amuniversal.com/bdc8a4d06d6401301d80001dd8b71c47",
			StripURL: ts.URL + "/strip/2000-01-01",
		},
		{
			Date:     "2018-10-30",
			Title:    "Intentionally Underbidding",
			ImageURL: "https://assets.amuniversal.com/cda546d0a88c01365b26005056a9545d",
			StripURL: ts.URL + "/strip/2018-10-30",
		},
		{
			Date:     "2019-11-02",
			Title:    "Multiple Choice",
			ImageURL: "https://assets.amuniversal.com/ce7ec130d6480137c832005056a9545d",
			StripURL: ts.URL + "/strip/2019-11-02",
		},
		{
			Date:     "2020-11-11",
			Title:    "Elbonian Words",
			ImageURL: "https://assets.amuniversal.com/f25312c0fb5b01382ef9005056a9545d",
			StripURL: ts.URL + "/strip/2020-11-11",
		},
	}

	for _, td := range testdata {
		comic, err := dilbert.ScrapeComic(context.Background(), td.Date)
		if err != nil {
			t.Error(err)
		}
		if diff := cmp.Diff(td, comic); diff != "" {
			t.Error(diff)
		}
	}
}
