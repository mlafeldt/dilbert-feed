package main

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestFeedGenerator(t *testing.T) {
	var buf bytes.Buffer
	date, _ := time.Parse("2006-01-02", "2018-10-01")
	g := FeedGenerator{
		BucketName: "dilbert-feed-example",
		StripsDir:  "strips/",
		StartDate:  date,
		FeedLength: 3,
	}

	if err := g.Generate(&buf); err != nil {
		t.Fatal(err)
	}

	got, err := ioutil.ReadAll(&buf)
	if err != nil {
		t.Fatal(err)
	}

	want, err := ioutil.ReadFile("testdata/feed.xml")
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Error(diff)
	}
}
