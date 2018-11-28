package epsagon

import (
	"fmt"
	"github.com/epsagon/epsagon-go/protocol"
	"runtime/debug"
	"time"
)

// GetTimestamp returns the current time in miliseconds
func GetTimestamp() float64 {
	return float64(time.Now().UnixNano()) / float64(time.Millisecond) / float64(time.Nanosecond) / 1000.0
}

// AddExceptionTypeAndMessage adds an exception to the current tracer with
// the current stack and time.
// exceptionType, msg are strings that will be added to the exception
func AddExceptionTypeAndMessage(exceptionType, msg string) {
	stack := debug.Stack()
	AddException(&protocol.Exception{
		Type:      exceptionType,
		Message:   msg,
		Traceback: string(stack),
		Time:      GetTimestamp(),
	})
}

// GeneralEpsagonRecover recover function that will send exception to epsagon
// exceptionType, msg are strings that will be added to the exception
func GeneralEpsagonRecover(exceptionType, msg string) {
	if r := recover(); r != nil {
		AddExceptionTypeAndMessage(exceptionType, fmt.Sprintf("%s:%+v", msg, r))
	}
}
