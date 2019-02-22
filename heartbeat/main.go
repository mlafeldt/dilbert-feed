package main

import (
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

type Input struct{}

type Output struct {
	Endpoint string `json:"endpoint"`
	Status   string `json:"status"`
}

func main() {
	lambda.Start(handler)
}

func handler(input Input) (*Output, error) {
	url := os.Getenv("HEARTBEAT_ENDPOINT")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "dilbert-feed")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &Output{
		Endpoint: url,
		Status:   resp.Status,
	}, nil
}
