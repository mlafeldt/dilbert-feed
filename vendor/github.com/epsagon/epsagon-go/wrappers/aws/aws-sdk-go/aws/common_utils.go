package epsagonawswrapper

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/epsagon/epsagon-go/epsagon"
	"github.com/epsagon/epsagon-go/protocol"
	"reflect"
	"strconv"
)

type specificOperationHandler func(r *request.Request, res *protocol.Resource, metadataOnly bool)

func handleSpecificOperation(
	r *request.Request,
	res *protocol.Resource,
	metadataOnly bool,
	handlers map[string]specificOperationHandler,
	defaultHandler specificOperationHandler,
) {
	handler := handlers[res.Operation]
	if handler == nil {
		handler = defaultHandler
	}
	if handler != nil {
		handler(r, res, metadataOnly)
	}
}

func getFieldStringPtr(value reflect.Value, fieldName string) (string, bool) {
	field := value.FieldByName(fieldName)
	if field == (reflect.Value{}) {
		return "", false
	}
	return field.Elem().String(), true
}

func updateMetadataFromBytes(
	value reflect.Value, fieldName string, targetKey string, metadata map[string]string) {
	field := value.FieldByName(fieldName)
	if field == (reflect.Value{}) {
		return
	}
	metadata[targetKey] = string(field.Bytes())
}

func updateMetadataFromValue(
	value reflect.Value, fieldName string, targetKey string, metadata map[string]string) {
	fieldValue, ok := getFieldStringPtr(value, fieldName)
	if ok {
		metadata[targetKey] = fieldValue
	}
}

func updateMetadataFromInt64(
	value reflect.Value, fieldName string, targetKey string, metadata map[string]string) {
	field := value.FieldByName(fieldName)
	if field == (reflect.Value{}) {
		return
	}
	metadata[targetKey] = strconv.FormatInt(field.Elem().Int(), 10)
}

func updateMetadataWithFieldToJSON(
	value reflect.Value, fieldName string, targetKey string, metadata map[string]string) {
	field := value.FieldByName(fieldName)
	if field == (reflect.Value{}) {
		return
	}
	stream, err := json.Marshal(field.Interface())
	if err != nil {
		epsagon.AddExceptionTypeAndMessage("aws-sdk-go", fmt.Sprintf("%v", err))
		return
	}
	metadata[targetKey] = string(stream)
}
