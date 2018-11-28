package epsagon

import (
	"bytes"
	"encoding/json"
	"fmt"
	lambdaEvents "github.com/aws/aws-lambda-go/events"
	lambdaContext "github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/epsagon/epsagon-go/protocol"
	"github.com/satori/go.uuid"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
)

type triggerFactory func(event interface{}, metadataOnly bool) *protocol.Event

func unknownTrigger(event interface{}, metadataOnly bool) *protocol.Event {
	return &protocol.Event{}
}

func getReflectType(i interface{}) reflect.Type {
	return reflect.TypeOf(i)
}

func mapParametersToString(params map[string]string) string {
	buf, err := json.Marshal(params)
	if err != nil {
		AddException(&protocol.Exception{
			Type:      "trigger-creation",
			Message:   fmt.Sprintf("Failed to serialize %v", params),
			Traceback: string(debug.Stack()),
			Time:      GetTimestamp(),
		})
		return ""
	}
	return string(buf)
}

func triggerAPIGatewayProxyRequest(rawEvent interface{}, metadataOnly bool) *protocol.Event {
	event, ok := rawEvent.(lambdaEvents.APIGatewayProxyRequest)
	if !ok {
		AddException(&protocol.Exception{
			Type: "trigger-creation",
			Message: fmt.Sprintf(
				"failed to convert rawEvent to lambdaEvents.APIGatewayProxyRequest %v",
				rawEvent),
			Time: GetTimestamp(),
		})
		return nil
	}
	triggerEvent := &protocol.Event{
		Id:        event.RequestContext.RequestID,
		Origin:    "trigger",
		StartTime: GetTimestamp(),
		Resource: &protocol.Resource{
			Name:      event.Resource,
			Type:      "api_gateway",
			Operation: event.HTTPMethod,
			Metadata: map[string]string{
				"stage":                   event.RequestContext.Stage,
				"query_string_parameters": mapParametersToString(event.QueryStringParameters),
				"path_parameters":         mapParametersToString(event.PathParameters),
			},
		},
	}
	if !metadataOnly {
		if bodyJSON, err := json.Marshal(event.Body); err != nil {
			AddException(&protocol.Exception{
				Type:      "trigger-creation",
				Message:   fmt.Sprintf("Failed to serialize body %s", event.Body),
				Traceback: string(debug.Stack()),
				Time:      GetTimestamp(),
			})
			triggerEvent.Resource.Metadata["body"] = ""
		} else {
			triggerEvent.Resource.Metadata["body"] = string(bodyJSON)
		}
		triggerEvent.Resource.Metadata["headers"] = mapParametersToString(event.Headers)
	}

	return triggerEvent
}

func triggerS3Event(rawEvent interface{}, metadataOnly bool) *protocol.Event {
	event, ok := rawEvent.(lambdaEvents.S3Event)
	if !ok {
		AddException(&protocol.Exception{
			Type: "trigger-creation",
			Message: fmt.Sprintf(
				"failed to convert rawEvent to lambdaEvents.S3Event %v",
				rawEvent),
			Time: GetTimestamp(),
		})
		return nil
	}

	triggerEvent := &protocol.Event{
		Id:         fmt.Sprintf("s3-trigger-%s", event.Records[0].ResponseElements["x-amz-request-id"]),
		Origin:    "trigger",
		StartTime: GetTimestamp(),
		Resource: &protocol.Resource{
			Name:      event.Records[0].S3.Bucket.Name,
			Type:      "s3",
			Operation: event.Records[0].EventName,
			Metadata: map[string]string{
				"region": event.Records[0].AWSRegion,
				"object_key": event.Records[0].S3.Object.Key,
				"object_size": strconv.FormatInt(event.Records[0].S3.Object.Size, 10),
				"object_etag": event.Records[0].S3.Object.ETag,
				"object_sequencer": event.Records[0].S3.Object.Sequencer,
				"x-amz-request-id": event.Records[0].ResponseElements["x-amz-request-id"],
			},
		},
	}

	return triggerEvent
}

func triggerKinesisEvent(rawEvent interface{}, metadataOnly bool) *protocol.Event {
	event, ok := rawEvent.(lambdaEvents.KinesisEvent)
	if !ok {
		AddException(&protocol.Exception{
			Type: "trigger-creation",
			Message: fmt.Sprintf(
				"failed to convert rawEvent to lambdaEvents.KinesisEvent %v",
				rawEvent),
			Time: GetTimestamp(),
		})
		return nil
	}

	eventSourceArnSlice := strings.Split(event.Records[0].EventSourceArn, "/")

	triggerEvent := &protocol.Event{
		Id:         event.Records[0].EventID,
		Origin:    "trigger",
		StartTime: GetTimestamp(),
		Resource: &protocol.Resource{
			Name:      eventSourceArnSlice[len(eventSourceArnSlice)-1],
			Type:      "kinesis",
			Operation: strings.Replace(event.Records[0].EventName, "aws:kinesis:", "", -1),
			Metadata: map[string]string{
				"region": event.Records[0].AwsRegion,
				"invoke_identity": event.Records[0].InvokeIdentityArn,
				"sequence_number": event.Records[0].Kinesis.SequenceNumber,
				"partition_key": event.Records[0].Kinesis.PartitionKey,
			},
		},
	}

	return triggerEvent
}

func triggerSNSEvent(rawEvent interface{}, metadataOnly bool) *protocol.Event {
	event, ok := rawEvent.(lambdaEvents.SNSEvent)
	if !ok {
		AddException(&protocol.Exception{
			Type: "trigger-creation",
			Message: fmt.Sprintf(
				"failed to convert rawEvent to lambdaEvents.SNSEvent %v",
				rawEvent),
			Time: GetTimestamp(),
		})
		return nil
	}

	eventSubscriptionArnSlice := strings.Split(event.Records[0].EventSubscriptionArn, ":")


	triggerEvent := &protocol.Event{
		Id:         event.Records[0].SNS.MessageID,
		Origin:    "trigger",
		StartTime: GetTimestamp(),
		Resource: &protocol.Resource{
			Name:      eventSubscriptionArnSlice[len(eventSubscriptionArnSlice)-2],
			Type:      "sns",
			Operation: event.Records[0].SNS.Type,
			Metadata: map[string]string{
				"Notification Subject": event.Records[0].SNS.Subject,
			},
		},
	}

	if !metadataOnly {
		triggerEvent.Resource.Metadata["Notification Message"] = event.Records[0].SNS.Message
	}

	return triggerEvent
}

func triggerSQSEvent(rawEvent interface{}, metadataOnly bool) *protocol.Event {
	event, ok := rawEvent.(lambdaEvents.SQSEvent)
	if !ok {
		AddException(&protocol.Exception{
			Type: "trigger-creation",
			Message: fmt.Sprintf(
				"failed to convert rawEvent to lambdaEvents.SQSEvent %v",
				rawEvent),
			Time: GetTimestamp(),
		})
		return nil
	}

	eventSourceArnSlice := strings.Split(event.Records[0].EventSourceARN, ":")

	triggerEvent := &protocol.Event{
		Id:         event.Records[0].MessageId,
		Origin:    "trigger",
		StartTime: GetTimestamp(),
		Resource: &protocol.Resource{
			Name:      eventSourceArnSlice[len(eventSourceArnSlice)-1],
			Type:      "sqs",
			Operation: "ReceiveMessage",
			Metadata: map[string]string{
				"MD5 Of Message Body": event.Records[0].Md5OfBody,
				"Sender ID": event.Records[0].Attributes["SenderId"],
				"Approximate Receive Count":
					event.Records[0].Attributes["ApproximateReceiveCount"],
				"Sent Timestamp":
					event.Records[0].Attributes["SentTimestamp"],
				"Approximate First Receive Timestamp":
					event.Records[0].Attributes["ApproximateFirstReceiveTimestamp"],
			},
		},
	}

	if !metadataOnly {
		triggerEvent.Resource.Metadata["Message Body"] = event.Records[0].Body
	}

	return triggerEvent
}

func triggerJSONEvent(rawEvent json.RawMessage, metadataOnly bool) *protocol.Event {
	triggerEvent := &protocol.Event{
		Id: uuid.NewV4().String(),
		Origin:    "trigger",
		StartTime: GetTimestamp(),
		Resource: &protocol.Resource{
			Name:      fmt.Sprintf("trigger-%s", lambdaContext.FunctionName),
			Type:      "json",
			Operation: "json",
			Metadata: map[string]string{
			},
		},
	}

	if !metadataOnly {
		triggerEvent.Resource.Metadata["data"] = string(rawEvent)
	}

	return triggerEvent
}

type factoryAndType struct {
	EventType reflect.Type
	Factory   triggerFactory
}

var (
	triggerFactories = map[string]factoryAndType{
		"api_gateway": {
			EventType: reflect.TypeOf(lambdaEvents.APIGatewayProxyRequest{}),
			Factory:   triggerAPIGatewayProxyRequest,
		},
		"aws:s3": {
			EventType: reflect.TypeOf(lambdaEvents.S3Event{}),
			Factory:   triggerS3Event,
		},
		"aws:kinesis": {
			EventType: reflect.TypeOf(lambdaEvents.KinesisEvent{}),
			Factory:   triggerKinesisEvent,
		},
		"aws:sns": {
			EventType: reflect.TypeOf(lambdaEvents.SNSEvent{}),
			Factory:   triggerSNSEvent,
		},
		"aws:sqs": {
			EventType: reflect.TypeOf(lambdaEvents.SQSEvent{}),
			Factory:   triggerSQSEvent,
		},
	}
)

func decodeAndUnpackEvent(
	payload json.RawMessage,
	eventType reflect.Type,
	factory triggerFactory,
	metadataOnly bool,
) *protocol.Event {

	event := reflect.New(eventType)
	decoder := json.NewDecoder(bytes.NewReader(payload))

	if err := decoder.Decode(event.Interface()); err != nil {
		// fmt.Printf("DEBUG: addLambdaTrigger error in json decoder: %v\n", err)
		return nil
	}
	return factory(event.Elem().Interface(), metadataOnly)
}

type recordField struct {
	EventSource string
}

type interestingFields struct {
	Records    []recordField
	HTTPMethod string
	Context    map[string]interface{}
	MethodArn  string
	Source     string
}

func guessTriggerSource(payload json.RawMessage) string {
	var rawEvent interestingFields
	err := json.Unmarshal(payload, &rawEvent)
	if err != nil {
		AddException(&protocol.Exception{
			Type:      "trigger-identification",
			Message:   fmt.Sprintf("Failed to unmarshal json %v\n", err),
			Traceback: string(debug.Stack()),
			Time:      GetTimestamp(),
		})
		return ""
	}
	triggerSource := "json"
	if len(rawEvent.Records) > 0 {
		triggerSource = rawEvent.Records[0].EventSource
	} else if len(rawEvent.HTTPMethod) > 0 {
		triggerSource = "api_gateway"
	} else if _, ok := rawEvent.Context["http-method"]; ok {
		triggerSource = "api_gateway_no_proxy"
	} else if len(rawEvent.Source) > 0 {
		sourceSlice := strings.Split(rawEvent.Source, ".")
		triggerSource = sourceSlice[len(sourceSlice)-1]
	}
	return triggerSource
}

func addLambdaTrigger(
	payload json.RawMessage,
	metadataOnly bool,
	triggerFactories map[string]factoryAndType,
	) {

	var triggerEvent *protocol.Event

	triggerSource := guessTriggerSource(payload)

	if triggerSource == "json" {
		triggerEvent = triggerJSONEvent(payload, metadataOnly)
	} else if triggerSource == "api_gateway_no_proxy" {
		// currently not supported, needs to extract data from json
	} else {
		factoryStruct, found := triggerFactories[triggerSource]
		if found {
			triggerEvent = decodeAndUnpackEvent(
				payload, factoryStruct.EventType, factoryStruct.Factory, metadataOnly)
		}
	}

	// If a trigger was found
	if triggerEvent != nil {
		AddEvent(triggerEvent)
	}
}
