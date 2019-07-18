package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

// Input is the input passed to the Lambda function.
type Input struct {
	Endpoint string `json:"endpoint"`
}

// Output is the output returned by the Lambda function.
type Output struct {
	Endpoint string `json:"endpoint"`
	Status   string `json:"status"`
}

func main() {
	lambda.Start(handler)
}

func handler(input Input) (*Output, error) {
	if input.Endpoint == "" {
		return nil, errors.New("endpoint must be set")
	}

	req, err := http.NewRequest("GET", input.Endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "dilbert-feed")

	client := http.Client{Timeout: 5 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &Output{
		Endpoint: input.Endpoint,
		Status:   resp.Status,
	}, nil
}
