package epsagon

import (
	"github.com/epsagon/epsagon-go/protocol"
)

// MockedEpsagonTracer will not send traces if closed
type MockedEpsagonTracer struct {
	Exceptions *[]*protocol.Exception
	Events     *[]*protocol.Event
	Config     *Config

	panicStart        bool
	panicAddEvent     bool
	panicAddException bool
	panicStop         bool
}

// Start implementes mocked Start
func (t *MockedEpsagonTracer) Start() {
	if t.panicStart {
		panic("panic in Start()")
	}
}

// Running implementes mocked Running
func (t *MockedEpsagonTracer) Running() bool {
	return false
}

// Stop implementes mocked Stop
func (t *MockedEpsagonTracer) Stop() {
	if t.panicStop {
		panic("panic in Stop()")
	}
}

// Stopped implementes mocked Stopped
func (t *MockedEpsagonTracer) Stopped() bool {
	return false
}

// AddEvent implementes mocked AddEvent
func (t *MockedEpsagonTracer) AddEvent(e *protocol.Event) {
	if t.panicAddEvent {
		panic("panic in AddEvent()")
	}
	*t.Events = append(*t.Events, e)
}

// AddException implementes mocked AddEvent
func (t *MockedEpsagonTracer) AddException(e *protocol.Exception) {
	if t.panicAddException {
		panic("panic in AddException()")
	}
	*t.Exceptions = append(*t.Exceptions, e)
}

// GetConfig implementes mocked AddEvent
func (t *MockedEpsagonTracer) GetConfig() *Config {
	return t.Config
}
