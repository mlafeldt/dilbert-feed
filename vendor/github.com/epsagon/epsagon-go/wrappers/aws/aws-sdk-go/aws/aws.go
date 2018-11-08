package epsagonawswrapper

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/epsagon/epsagon-go/epsagon"
	"github.com/epsagon/epsagon-go/protocol"
	"log"
	"strings"
	"time"
)

// WrapSession wraps an aws session.Session with epsgaon traces
func WrapSession(s *session.Session) *session.Session {
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

type extractor func(*request.Request, *protocol.Resource)

var awsResourceDataExtractors = map[string]map[string]extractor{
	"sqs": map[string]extractor{
		"SendMessage": sqsSendMessageExtractor,
		"GetQueueUrl": sqsGetQueueURLExtractor,
	},
}

func extractResourceInformation(r *request.Request) *protocol.Resource {
	res := protocol.Resource{
		Type:      r.ClientInfo.ServiceName,
		Operation: r.Operation.Name,
		Metadata:  make(map[string]string),
	}
	extractor := awsResourceDataExtractors[res.Type][res.Operation]
	if extractor != nil {
		extractor(r, &res)
	} else {
		defaultExtractor(r, &res)
	}
	return &res
}

func defaultExtractor(r *request.Request, res *protocol.Resource) {
	if epsagon.GetGlobalTracerConfig().Debug {
		log.Println("EPSAGON DEBUG:: entering defaultExtractor")
	}
	extractInterfaceToMetadata(r.Data, res)
	extractInterfaceToMetadata(r.Params, res)
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

func sqsSendMessageExtractor(r *request.Request, res *protocol.Resource) {
	input, ok := r.Params.(*sqs.SendMessageInput)
	if !ok {
		log.Printf("EPSAGON DEBUG: sqsSendMessageExtractor failed to unmarshal data: r.Params %+v", r.Params)
		defaultExtractor(r, res)
		return
	}
	urlParts := strings.Split(*input.QueueUrl, "/")
	res.Name = urlParts[len(urlParts)-1]
	output, ok := r.Data.(*sqs.SendMessageOutput)
	if !ok {
		log.Printf("EPSAGON DEBUG: sqsSendMessageExtractor failed to unmarshal data: r.Data %+v", r.Data)
		defaultExtractor(r, res)
		return
	}
	res.Metadata["Message ID"] = *output.MessageId
	res.Metadata["MD5 Of Message Body"] = *output.MD5OfMessageBody
}

func sqsGetQueueURLExtractor(r *request.Request, res *protocol.Resource) {
	input, ok := r.Params.(*sqs.GetQueueUrlInput)
	if !ok {
		log.Printf("EPSAGON DEBUG: sqsGetQueueURLExtractor failed to unmarshal data: r.Params %+v", r.Params)
		defaultExtractor(r, res)
		return
	}
	res.Name = *input.QueueName
}
