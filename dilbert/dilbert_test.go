package dilbert

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewComic(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()
	baseURL = ts.URL

	testdata := []*Comic{
		{
			Date:     "2000-01-01",
			Title:    "",
			ImageURL: "https://assets.amuniversal.com/bdc8a4d06d6401301d80001dd8b71c47",
			StripURL: baseURL + "/strip/2000-01-01",
		},
		{
			Date:     "2018-10-30",
			Title:    "Intentionally Underbidding",
			ImageURL: "https://assets.amuniversal.com/cda546d0a88c01365b26005056a9545d",
			StripURL: baseURL + "/strip/2018-10-30",
		},
	}

	for _, td := range testdata {
		comic, err := NewComic(td.Date)
		if err != nil {
			t.Error(err)
		}
		if diff := cmp.Diff(td, comic); diff != "" {
			t.Error(diff)
		}
	}
}
