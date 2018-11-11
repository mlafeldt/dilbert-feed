package epsagonawswrapper

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/epsagon/epsagon-go/epsagon"
	"github.com/epsagon/epsagon-go/protocol"
	"log"
	"time"
)

// WrapSession wraps an aws session.Session with epsgaon traces
func WrapSession(s *session.Session) *session.Session {
	if s == nil {
		return s
	}
	s.Handlers.Complete.PushFrontNamed(
		request.NamedHandler{
			Name: "github.com/epsagon/epsagon-go/wrappers/aws/aws-sdk-go/aws/aws.go",
			Fn:   completeEventData,
		})
	return s
}

func getTimeStampFromRequest(r *request.Request) float64 {
	return float64(r.Time.UTC().UnixNano()) / float64(time.Millisecond) / float64(time.Nanosecond) / 1000.0
}

func completeEventData(r *request.Request) {
	defer epsagon.GeneralEpsagonRecover("aws-sdk-go wrapper", "")
	if epsagon.GetGlobalTracerConfig().Debug {
		log.Printf("EPSAGON DEBUG OnComplete request response: %+v\n", r.HTTPResponse)
		log.Printf("EPSAGON DEBUG OnComplete request Operation: %+v\n", r.Operation)
		log.Printf("EPSAGON DEBUG OnComplete request ClientInfo: %+v\n", r.ClientInfo)
		log.Printf("EPSAGON DEBUG OnComplete request Params: %+v\n", r.Params)
		log.Printf("EPSAGON DEBUG OnComplete request Data: %+v\n", r.Data)
	}

	endTime := epsagon.GetTimestamp()
	event := protocol.Event{
		Id:        r.RequestID,
		StartTime: getTimeStampFromRequest(r),
		Origin:    "aws-sdk",
		Resource:  extractResourceInformation(r),
	}
	event.Duration = endTime - event.StartTime
	epsagon.AddEvent(&event)
}

type factory func(*request.Request, *protocol.Resource, bool)

var awsResourceEventFactories = map[string]factory{
	"sqs":      sqsEventDataFactory,
	"s3":       s3EventDataFactory,
	"dynamodb": dynamodbEventDataFactory,
	"kinesis":  kinesisEventDataFactory,
	"ses":      sesEventDataFactory,
	"sns":      snsEventDataFactory,
	"lambda":   lambdaEventDataFactory,
	"sfn":      sfnEventDataFactory,
}

func extractResourceInformation(r *request.Request) *protocol.Resource {
	res := protocol.Resource{
		Type:      r.ClientInfo.ServiceName,
		Operation: r.Operation.Name,
		Metadata:  make(map[string]string),
	}
	factory := awsResourceEventFactories[res.Type]
	if factory != nil {
		factory(r, &res, epsagon.GetGlobalTracerConfig().MetadataOnly)
	} else {
		defaultFactory(r, &res, epsagon.GetGlobalTracerConfig().MetadataOnly)
	}
	return &res
}

func defaultFactory(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	if epsagon.GetGlobalTracerConfig().Debug {
		log.Println("EPSAGON DEBUG:: entering defaultFactory")
	}
	if !metadataOnly {
		extractInterfaceToMetadata(r.Data, res)
		extractInterfaceToMetadata(r.Params, res)
	}
}

func extractInterfaceToMetadata(input interface{}, res *protocol.Resource) {
	var data map[string]interface{}
	rawJSON, err := json.Marshal(input)
	if err != nil {
		log.Printf("EPSAGON DEBUG: Failed to marshal input: %+v\n", input)
		return
	}
	err = json.Unmarshal(rawJSON, &data)
	if err != nil {
		log.Printf("EPSAGON DEBUG: Failed to unmarshal input: %+v\n", rawJSON)
		return
	}
	for key, value := range data {
		res.Metadata[key] = fmt.Sprintf("%v", value)
	}
}
