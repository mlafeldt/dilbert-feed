package epsagonawswrapper

import (
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/epsagon/epsagon-go/protocol"
	"reflect"
)

func sesEventDataFactory(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	handleSpecificOperation(r, res, metadataOnly,
		map[string]specificOperationHandler{
			"SendEmail": handleSESSendEmail,
		},
		nil,
	)
}

func handleSESSendEmail(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	inputValue := reflect.ValueOf(r.Params).Elem()
	updateMetadataFromValue(inputValue, "Source", "source", res.Metadata)
	updateMetadataWithFieldToJSON(inputValue, "Destination", "destination", res.Metadata)
	messageField := inputValue.FieldByName("Message")
	if messageField != (reflect.Value{}) {
		updateMetadataWithFieldToJSON(messageField, "Subject", "subject", res.Metadata)
		if !metadataOnly {
			updateMetadataWithFieldToJSON(messageField, "Body", "body", res.Metadata)
		}
	}
	outputValue := reflect.ValueOf(r.Data).Elem()
	updateMetadataFromValue(outputValue, "MessageId", "message_id", res.Metadata)
}
