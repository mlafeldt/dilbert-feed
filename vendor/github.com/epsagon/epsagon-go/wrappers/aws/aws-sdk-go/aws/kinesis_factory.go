package epsagonawswrapper

import (
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/epsagon/epsagon-go/epsagon"
	"github.com/epsagon/epsagon-go/protocol"
	"reflect"
)

func kinesisEventDataFactory(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	inputValue := reflect.ValueOf(r.Params).Elem()
	streamName, ok := getFieldStringPtr(inputValue, "StreamName")
	if !ok {
		epsagon.AddExceptionTypeAndMessage("aws-sdk-go",
			"kinesisEventDataFactory: couldn't find StreamName")
	}
	res.Name = streamName
	updateMetadataFromValue(inputValue, "PartitionKey", "partition_key", res.Metadata)
	if !metadataOnly {
		dataField := inputValue.FieldByName("Data")
		if dataField != (reflect.Value{}) {
			res.Metadata["data"] = string(dataField.Bytes())
		}
	}
	handleSpecificOperation(r, res, metadataOnly,
		map[string]specificOperationHandler{
			"PutRecord": handleKinesisPutRecord,
		}, nil,
	)
}

func handleKinesisPutRecord(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	outputValue := reflect.ValueOf(r.Data).Elem()
	updateMetadataFromValue(outputValue, "ShardId", "shared_id", res.Metadata)
	updateMetadataFromValue(outputValue, "SequenceNumber", "sequence_number", res.Metadata)
}
