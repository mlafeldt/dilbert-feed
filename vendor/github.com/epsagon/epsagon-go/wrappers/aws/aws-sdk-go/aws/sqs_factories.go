package epsagonawswrapper

import (
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/epsagon/epsagon-go/protocol"
	"reflect"
	"strconv"
	"strings"
)

func sqsEventDataFactory(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	inputValue := reflect.ValueOf(r.Params).Elem()
	identifyQueueName(inputValue, res)

	if !metadataOnly {
		updateMessageBody(inputValue, res)
	}

	handleSpecificOperations := map[string]specificOperationHandler{
		"SendMessage":    handleSQSSendMessage,
		"ReceiveMessage": handleSQSReceiveMessage,
	}
	handler := handleSpecificOperations[res.Operation]
	if handler != nil {
		handler(r, res, metadataOnly)
	}
}

func updateMessageBody(inputValue reflect.Value, res *protocol.Resource) {
	entry := inputValue.FieldByName("Entries")
	if entry == (reflect.Value{}) || entry.Len() == 0 {
		entry = inputValue
	} else {
		// TODO currently only records the first message
		entry = entry.Index(0).Elem()
	}
	updateMetadataFromValue(entry, "MessageBody", "MessageBody", res.Metadata)
}

func identifyQueueName(inputValue reflect.Value, res *protocol.Resource) {
	queueURLField := inputValue.FieldByName("QueueUrl")
	if queueURLField != (reflect.Value{}) {
		queueURL, ok := queueURLField.Elem().Interface().(string)
		if ok {
			urlParts := strings.Split(queueURL, "/")
			res.Name = urlParts[len(urlParts)-1]
		} // TODO else send exception?
	} else {
		queueNameField := inputValue.FieldByName("QueueName")
		if queueNameField != (reflect.Value{}) {
			res.Name = queueNameField.Elem().String()
		}
	}
}

func handleSQSSendMessage(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	outputValue := reflect.ValueOf(r.Data).Elem()
	updateMetadataFromValue(outputValue, "MessageId", "Message ID", res.Metadata)
	updateMetadataFromValue(outputValue, "MD5OfMessageBodyMessageId",
		"MD5 Of Message Body", res.Metadata)
}

func handleSQSReceiveMessage(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	var numberOfMessages int
	outputValue := reflect.ValueOf(r.Data).Elem()
	messages := outputValue.FieldByName("Messages")
	if messages == (reflect.Value{}) {
		numberOfMessages = 0
	} else {
		numberOfMessages = messages.Len()
		if numberOfMessages > 0 {
			updateMetadataFromValue(messages.Index(0).Elem(), "MessageId", "Message ID", res.Metadata)
			updateMetadataFromValue(messages.Index(0).Elem(), "MD5OfMessageBodyMessageId",
				"MD5 Of Message Body", res.Metadata)
		}
	}
	res.Metadata["Number Of Messages"] = strconv.Itoa(numberOfMessages)
}
