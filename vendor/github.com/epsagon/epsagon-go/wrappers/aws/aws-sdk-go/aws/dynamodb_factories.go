package epsagonawswrapper

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/epsagon/epsagon-go/epsagon"
	"github.com/epsagon/epsagon-go/protocol"
	"reflect"
)

func dynamodbEventDataFactory(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	inputValue := reflect.ValueOf(r.Params).Elem()
	tableName, ok := getFieldStringPtr(inputValue, "TableName")
	if ok {
		res.Name = tableName
	}
	handleSpecificOperations := map[string]specificOperationHandler{
		"PutItem":        handleDynamoDBPutItem,
		"GetItem":        handleDynamoDBGetItem,
		"DeleteItem":     handleDynamoDBDeleteItem,
		"UpdateItem":     handleDynamoDBUpdateItem,
		"Scan":           handleDynamoDBScan,
		"BatchWriteItem": handleDynamoDBBatchWriteItem,
	}
	handler := handleSpecificOperations[res.Operation]
	if handler != nil {
		handler(r, res, metadataOnly)
	}
}

func deserializeAttributeMap(inputField reflect.Value) map[string]string {
	formattedItem := make(map[string]string)
	input := inputField.Interface().(map[string]*dynamodb.AttributeValue)
	for k, v := range input {
		formattedItem[k] = v.String()
	}
	return formattedItem
}

func jsonAttributeMap(inputField reflect.Value) string {
	if inputField == (reflect.Value{}) {
		return ""
	}
	formattedMap := deserializeAttributeMap(inputField)
	stream, err := json.Marshal(formattedMap)
	if err != nil {
		epsagon.AddExceptionTypeAndMessage("aws-sdk-go", fmt.Sprintf("%v", err))
		return ""
	}
	return string(stream)
}

func handleDynamoDBPutItem(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	inputValue := reflect.ValueOf(r.Params).Elem()
	itemField := inputValue.FieldByName("Item")
	if itemField == (reflect.Value{}) {
		return
	}
	formattedItem := deserializeAttributeMap(itemField)
	formattedItemStream, err := json.Marshal(formattedItem)
	if err != nil {
		// TODO send tracer exception?
		return
	}
	if !metadataOnly {
		res.Metadata["Item"] = string(formattedItemStream)
	}
	h := md5.New()
	h.Write(formattedItemStream)
	res.Metadata["item_hash"] = hex.EncodeToString(h.Sum(nil))
}

func handleDynamoDBGetItem(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	inputValue := reflect.ValueOf(r.Params).Elem()
	jsonKeyField := jsonAttributeMap(inputValue.FieldByName("Key"))
	res.Metadata["Key"] = jsonKeyField

	if !metadataOnly {
		outputValue := reflect.ValueOf(r.Data).Elem()
		jsonItemField := jsonAttributeMap(outputValue.FieldByName("Item"))
		res.Metadata["Item"] = jsonItemField
	}
}

func handleDynamoDBDeleteItem(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	inputValue := reflect.ValueOf(r.Params).Elem()
	jsonKeyField := jsonAttributeMap(inputValue.FieldByName("Key"))
	res.Metadata["Key"] = jsonKeyField
}

func handleDynamoDBUpdateItem(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	inputValue := reflect.ValueOf(r.Params).Elem()
	eavField := inputValue.FieldByName("ExpressionAttributeValues")
	eav := deserializeAttributeMap(eavField)
	eavStream, err := json.Marshal(eav)
	if err != nil {
		return
	}
	updateParameters := map[string]string{
		"Expression Attribute Values": string(eavStream),
	}
	jsonKeyField := jsonAttributeMap(inputValue.FieldByName("Key"))
	updateParameters["Key"] = jsonKeyField
	updateMetadataFromValue(inputValue,
		"UpdateExpression", "UpdateExpression", updateParameters)
	updateParamsStream, err := json.Marshal(updateParameters)
	if err != nil {
		return
	}
	res.Metadata["Update Parameters"] = string(updateParamsStream)
}

func deserializeItems(itemsField reflect.Value) string {
	if itemsField == (reflect.Value{}) {
		return ""
	}
	formattedItems := make([]map[string]string, itemsField.Len())
	for ind := 0; ind < itemsField.Len(); ind++ {
		formattedItems = append(formattedItems,
			deserializeAttributeMap(itemsField.Index(ind)))
	}
	formattedItemsStream, err := json.Marshal(formattedItems)
	if err != nil {
		epsagon.AddExceptionTypeAndMessage("aws-sdk-go",
			fmt.Sprintf("sederializeItems: %v", err))
	}
	return string(formattedItemsStream)
}

func handleDynamoDBScan(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	outputValue := reflect.ValueOf(r.Params).Elem()
	updateMetadataFromInt64(outputValue, "Count", "Items Count", res.Metadata)
	updateMetadataFromInt64(outputValue, "ScannedCount", "Scanned Items Count", res.Metadata)
	itemsField := outputValue.FieldByName("Items")
	if !metadataOnly {
		res.Metadata["Items"] = deserializeItems(itemsField)
	}
}

func handleDynamoDBBatchWriteItem(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	inputValue := reflect.ValueOf(r.Params).Elem()
	requestItemsField := inputValue.FieldByName("RequestItems")
	if requestItemsField != (reflect.Value{}) {
		var tableName string
		requestItems, ok := requestItemsField.Interface().(map[string][]*dynamodb.WriteRequest)
		if !ok {
			epsagon.AddExceptionTypeAndMessage("aws-sdk-go",
				"handleDynamoDBBatchWriteItem: Failed to cast RequestItems")
			return
		}
		for k := range requestItems {
			tableName = k
			break
		}
		res.Name = tableName
		// TODO not ignore other tables
		if !metadataOnly {
			items := make([]map[string]*dynamodb.AttributeValue, len(requestItems))
			for _, writeRequest := range requestItems[tableName] {
				items = append(items, writeRequest.PutRequest.Item)
			}
			itemsValue := reflect.ValueOf(items)
			res.Metadata["Items"] = deserializeItems(itemsValue)
		}
	}
}
