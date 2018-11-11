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
func GeneralEpsagonRecover(exceptionType, msg string) {
	if r := recover(); r != nil {
		AddExceptionTypeAndMessage(exceptionType, fmt.Sprintf("%s:%+v", msg, r))
	}
}

// MockedEpsagonTracer will not send traces if closed
type MockedEpsagonTracer struct {
	Exceptions *[]*protocol.Exception
	Events     *[]*protocol.Event
	Config     *Config
}

// Run implementes mocked Run
func (t *MockedEpsagonTracer) Run() {}

// Running implementes mocked Running
func (t *MockedEpsagonTracer) Running() bool {
	return false
}

// Stop implementes mocked Stop
func (t *MockedEpsagonTracer) Stop() {}

// Stopped implementes mocked Stopped
func (t *MockedEpsagonTracer) Stopped() bool {
	return false
}

// AddEvent implementes mocked AddEvent
func (t *MockedEpsagonTracer) AddEvent(e *protocol.Event) {
	*t.Events = append(*t.Events, e)
}

// AddException implementes mocked AddEvent
func (t *MockedEpsagonTracer) AddException(e *protocol.Exception) {
	*t.Exceptions = append(*t.Exceptions, e)
}

// GetConfig implementes mocked AddEvent
func (t *MockedEpsagonTracer) GetConfig() *Config {
	return t.Config
}
