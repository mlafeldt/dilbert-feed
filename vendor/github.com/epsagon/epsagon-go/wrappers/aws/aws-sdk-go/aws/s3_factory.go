package epsagonawswrapper

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/epsagon/epsagon-go/protocol"
	"reflect"
	"strings"
	"time"
)

func s3EventDataFactory(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	inputValue := reflect.ValueOf(r.Params).Elem()
	bucketName, ok := getFieldStringPtr(inputValue, "Bucket")
	if !ok {
		res.Name = bucketName
	}
	handleSpecificOperations := map[string]specificOperationHandler{
		"HeadObject":  handleS3GetOrHeadObject,
		"GetObject":   handleS3GetOrHeadObject,
		"PutObject":   handleS3PutObject,
		"ListObjects": handleS3ListObject,
	}
	handler := handleSpecificOperations[res.Operation]
	if handler != nil {
		handler(r, res, metadataOnly)
	}
}

func commonS3OpertionHandler(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	inputValue := reflect.ValueOf(r.Params).Elem()
	updateMetadataFromValue(inputValue, "Key", "key", res.Metadata)
	outputValue := reflect.ValueOf(r.Data).Elem()
	etag, ok := getFieldStringPtr(outputValue, "ETag")
	if ok {
		etag = strings.Trim(etag, "\"")
		res.Metadata["etag"] = etag
	}
}

func handleS3GetOrHeadObject(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	commonS3OpertionHandler(r, res, metadataOnly)
	outputValue := reflect.ValueOf(r.Data).Elem()
	updateMetadataFromValue(outputValue, "ContentLength", "file_size", res.Metadata)

	lastModifiedField := outputValue.FieldByName("LastModified")
	if lastModifiedField == (reflect.Value{}) {
		return
	}
	lastModified := lastModifiedField.Elem().Interface().(time.Time)
	res.Metadata["last_modified"] = lastModified.String()
}

func handleS3PutObject(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	commonS3OpertionHandler(r, res, metadataOnly)
}

type s3File struct {
	key  string
	size int64
	etag string
}

func handleS3ListObject(r *request.Request, res *protocol.Resource, metadataOnly bool) {
	if metadataOnly {
		return
	}

	outputValue := reflect.ValueOf(r.Data).Elem()
	contentsField := outputValue.FieldByName("Contents")
	if contentsField == (reflect.Value{}) {
		return
	}
	length := contentsField.Len()
	files := make([]s3File, length)
	for i := 0; i < length; i++ {
		var key, etag string
		var size int64
		fileObject := contentsField.Index(i).Elem()
		etag = fileObject.FieldByName("ETag").Elem().String()
		key = fileObject.FieldByName("Key").Elem().String()
		size = fileObject.FieldByName("Size").Elem().Int()

		files = append(files, s3File{key, size, etag})
	}
	res.Metadata["files"] = fmt.Sprintf("%+v", files)
}
