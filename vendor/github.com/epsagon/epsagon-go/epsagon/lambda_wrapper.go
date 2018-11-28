package epsagon

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/epsagon/epsagon-go/protocol"
	"os"
	"runtime/debug"
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

type invocationData struct {
	ExceptionInfo *protocol.Exception
	errorStatus protocol.ErrorCode
	result interface{}
	err error
	thrownError interface{}
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
	metadata := map[string]string{}
	lc, ok := lambdacontext.FromContext(ctx)
	if !ok {
		lc = &lambdacontext.LambdaContext{}
	}
	defer func() {
		if r := recover(); r != nil {
			AddExceptionTypeAndMessage("LambdaWrapper",
				fmt.Sprintf("preInvokeOps:%+v", r))
			info = &preInvokeData{
				LambdaContext:      lc,
				StartTime:          startTime,
				InvocationMetadata: metadata,
			}
		}
	}()

	metadata = map[string]string{
		"log_stream_name":  lambdacontext.LogStreamName,
		"log_group_name":   lambdacontext.LogGroupName,
		"function_version": lambdacontext.FunctionVersion,
		"memory":           strconv.Itoa(lambdacontext.MemoryLimitInMB),
		"cold_start":       strconv.FormatBool(coldStart),
		"aws_account":      getAWSAccount(lc),
		"region":           os.Getenv("AWS_REGION"),
	}
	coldStart = false

	addLambdaTrigger(payload, handler.config.MetadataOnly, triggerFactories)

	return &preInvokeData{
		InvocationMetadata: metadata,
		LambdaContext:      lc,
		StartTime:          startTime,
	}
}

func (handler *epsagonLambdaWrapper) postInvokeOps(
	preInvokeInfo *preInvokeData,
	invokeInfo *invocationData) {
	defer func() {
		if r := recover(); r != nil {
			AddExceptionTypeAndMessage("LambdaWrapper", fmt.Sprintf("postInvokeOps:%+v", r))
		}
	}()

	endTime := GetTimestamp()
	duration := endTime - preInvokeInfo.StartTime

	lambdaEvent := &protocol.Event{
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
		ErrorCode: invokeInfo.errorStatus,
		Exception: invokeInfo.ExceptionInfo,
	}

	if !handler.config.MetadataOnly {
		lambdaEvent.Resource.Metadata["return_value"] = fmt.Sprintf("%+v", invokeInfo.result)
	}

	AddEvent(lambdaEvent)
}

// Invoke calls the handler, and creates a tracer for that duration.
func (handler *epsagonLambdaWrapper) Invoke(ctx context.Context, payload json.RawMessage) (result interface{}, err error) {
	invokeInfo := &invocationData {}
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
		if invokeInfo.thrownError != nil {
			panic(invokeInfo.thrownError)
		}
	}()

	handler.createTracer()
	defer StopTracer()

	preInvokeInfo := handler.preInvokeOps(ctx, payload)
	handler.InvokeClientLambda(ctx, payload, invokeInfo)
	handler.postInvokeOps(preInvokeInfo, invokeInfo)

	return invokeInfo.result, invokeInfo.err
}

func (handler *epsagonLambdaWrapper) InvokeClientLambda(
	ctx context.Context, payload json.RawMessage, invokeInfo *invocationData) {
	defer func() {
		invokeInfo.thrownError = recover()
		if invokeInfo.thrownError != nil {
			invokeInfo.ExceptionInfo = &protocol.Exception{
				Type: "Runtime Error",
				Message: fmt.Sprintf("%v", invokeInfo.thrownError),
				Traceback: string(debug.Stack()),
				Time:      GetTimestamp(),
			}
			invokeInfo.errorStatus = protocol.ErrorCode_EXCEPTION
		}
	}()

	invokeInfo.errorStatus = protocol.ErrorCode_OK
	// calling the actual function:
	handler.invoked = true
	handler.invoking = true
	result, err := handler.handler(ctx, payload)
	handler.invoking = false
	if err != nil {
		invokeInfo.errorStatus = protocol.ErrorCode_ERROR
	}
	invokeInfo.result = result
	invokeInfo.err = err
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
