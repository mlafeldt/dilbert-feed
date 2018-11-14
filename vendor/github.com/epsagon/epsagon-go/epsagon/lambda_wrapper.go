package epsagon

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/epsagon/epsagon-go/protocol"
	"os"
	"strconv"
	"strings"
)

var (
	coldStart = true
)

type genericHandler func(context.Context, json.RawMessage) (interface{}, error)

// epsagonLambdaWrapper is a generic lambda function type
type epsagonLambdaWrapper struct {
	handler  genericHandler
	config   *Config
	invoked  bool
	invoking bool
}

func (handler *epsagonLambdaWrapper) createTracer() {
	if handler.config == nil {
		handler.config = &Config{}
	}
	CreateTracer(handler.config)
}

type preInvokeData struct {
	InvocationMetadata map[string]string
	LambdaContext      *lambdacontext.LambdaContext
	StartTime          float64
}

func getAWSAccount(lc *lambdacontext.LambdaContext) string {
	arnParts := strings.Split(lc.InvokedFunctionArn, ":")
	if len(arnParts) >= 4 {
		return arnParts[4]
	}
	return ""
}

func (handler *epsagonLambdaWrapper) preInvokeOps(
	ctx context.Context, payload json.RawMessage) (info *preInvokeData) {
	startTime := GetTimestamp()
	lc, ok := lambdacontext.FromContext(ctx)
	if !ok {
		lc = &lambdacontext.LambdaContext{}
	}
	defer func() {
		if r := recover(); r != nil {
			AddExceptionTypeAndMessage("LambdaWrapper",
				fmt.Sprintf("preInvokeOps:%+v", r))
		}
		info = &preInvokeData{
			LambdaContext:      lc,
			StartTime:          startTime,
			InvocationMetadata: map[string]string{},
		}
	}()

	addLambdaTrigger(payload, handler.config.MetadataOnly, triggerFactories)

	metadata := map[string]string{
		"log_stream_name":  lambdacontext.LogStreamName,
		"log_group_name":   lambdacontext.LogGroupName,
		"function_version": lambdacontext.FunctionVersion,
		"memory":           strconv.Itoa(lambdacontext.MemoryLimitInMB),
		"cold_start":       strconv.FormatBool(coldStart),
		"aws_account":      getAWSAccount(lc),
		"region":           os.Getenv("AWS_REGION"),
	}
	coldStart = false

	return &preInvokeData{
		InvocationMetadata: metadata,
		LambdaContext:      lc,
		StartTime:          startTime,
	}
}

func (handler *epsagonLambdaWrapper) postInvokeOps(errorStatus protocol.ErrorCode, preInvokeInfo *preInvokeData) {
	defer func() {
		if r := recover(); r != nil {
			AddExceptionTypeAndMessage("LambdaWrapper", fmt.Sprintf("postInvokeOps:%+v", r))
		}
	}()

	endTime := GetTimestamp()
	duration := endTime - preInvokeInfo.StartTime
	AddEvent(&protocol.Event{
		Id:        preInvokeInfo.LambdaContext.AwsRequestID,
		StartTime: preInvokeInfo.StartTime,
		Resource: &protocol.Resource{
			Name:      lambdacontext.FunctionName,
			Type:      "lambda",
			Operation: "invoke",
			Metadata:  preInvokeInfo.InvocationMetadata,
		},
		Origin:    "runner",
		Duration:  duration,
		ErrorCode: errorStatus,
	})
}

// Invoke calls the handler, and creates a tracer for that duration.
func (handler *epsagonLambdaWrapper) Invoke(ctx context.Context, payload json.RawMessage) (result interface{}, err error) {
	handler.invoked = false
	handler.invoking = false
	defer func() {
		if !handler.invoking {
			recover()
			// In the future might attempt to send basic data
		}
		if !handler.invoked {
			result, err = handler.handler(ctx, payload)
		}
	}()

	handler.createTracer()
	defer StopTracer()
	preInvokeInfo := handler.preInvokeOps(ctx, payload)

	errorStatus := protocol.ErrorCode_OK
	// calling the actual function:
	handler.invoked = true
	handler.invoking = true
	result, err = handler.handler(ctx, payload)
	handler.invoking = false
	if err != nil {
		errorStatus = protocol.ErrorCode_ERROR
	}

	handler.postInvokeOps(errorStatus, preInvokeInfo)

	return result, err
}

// WrapLambdaHandler wraps a generic handler for lambda function with epsagon tracing
func WrapLambdaHandler(config *Config, handler interface{}) interface{} {
	return func(ctx context.Context, payload json.RawMessage) (interface{}, error) {
		wrapper := &epsagonLambdaWrapper{
			config:  config,
			handler: makeGenericHandler(handler),
		}
		return wrapper.Invoke(ctx, payload)
	}
}
