package epsagon

import (
	"bytes"
	"fmt"
	"github.com/epsagon/epsagon-go/protocol"
	"github.com/golang/protobuf/jsonpb"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	mutex sync.Mutex
	// GlobalTracer A global Tracer for all internal uses
	GlobalTracer Tracer
)

// Tracer is what a general program tracer had to provide
type Tracer interface {
	AddEvent(*protocol.Event)
	AddException(*protocol.Exception)
	Start()
	Running() bool
	Stop()
	Stopped() bool
	GetConfig() *Config
}

// Config is the configuration for Epsagon's tracer
type Config struct {
	ApplicationName string
	Token           string
	CollectorURL    string
	MetadataOnly    bool
	Debug           bool
}

type epsagonTracer struct {
	Config *Config

	eventsPipe     chan *protocol.Event
	events         []*protocol.Event
	exceptionsPipe chan *protocol.Exception
	exceptions     []*protocol.Exception

	closeCmd chan struct{}
	stopped  chan struct{}
	running  chan struct{}
}

// Start starts running the tracer in another goroutine and returns
// when it is ready, or after 1 second timeout
func (tracer *epsagonTracer) Start() {
	go tracer.Run()
	timer := time.NewTimer(time.Second)
	select {
	case <-tracer.running:
		return
	case <-timer.C:
		log.Println("Epsagon Tracer couldn't start after one second timeout")
	}
}

func (tracer *epsagonTracer) sendTraces() {
	tracesReader, err := tracer.getTraceReader()
	if err != nil {
		// TODO create an exception and send a trace only with that
		log.Printf("Epsagon: Encountered an error while marshaling the traces: %v\n", err)
		return
	}
	client := &http.Client{Timeout: time.Duration(time.Second)}

	resp, err := client.Post(tracer.Config.CollectorURL, "application/json", tracesReader)
	if err != nil {
		var respBody []byte
		resp.Body.Read(respBody)
		resp.Body.Close()
		log.Printf("Error while sending traces \n%v\n%+v\n", err, respBody)
	}
}

func (tracer *epsagonTracer) getTraceReader() (io.Reader, error) {
	version := "go " + runtime.Version()
	trace := protocol.Trace{
		AppName:    tracer.Config.ApplicationName,
		Token:      tracer.Config.Token,
		Events:     tracer.events,
		Exceptions: tracer.exceptions,
		Version:    "1.0.0",
		Platform:   version,
	}
	if tracer.Config.Debug {
		log.Printf("EPSAGON DEBUG sending trace: %+v\n", trace)
	}

	marshaler := jsonpb.Marshaler{
		EnumsAsInts: true, EmitDefaults: true, OrigName: true}
	traceJSON, err := marshaler.MarshalToString(&trace)
	if err != nil {
		return nil, err
	}
	if tracer.Config.Debug {
		log.Printf("Final Traces: %s ", traceJSON)
	}
	return bytes.NewBuffer([]byte(traceJSON)), nil
}

func isChannelPinged(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// Running return true iff the tracer has been running
func (tracer *epsagonTracer) Running() bool {
	return isChannelPinged(tracer.running)
}

// Stopped return true iff the tracer has been closed
func (tracer *epsagonTracer) Stopped() bool {
	return isChannelPinged(tracer.stopped)
}

func fillConfigDefaults(config *Config) {
	if !config.Debug {
		if os.Getenv("EPSAGON_DEBUG") == "TRUE" {
			config.Debug = true
		}
	}
	if len(config.Token) == 0 {
		config.Token = os.Getenv("EPSAGON_TOKEN")
		if config.Debug {
			log.Println("EPSAGON DEBUG: setting token from environment variable")
		}
	}
	if len(config.CollectorURL) == 0 {
		envURL := os.Getenv("EPSAGON_COLLECTOR_URL")
		if len(envURL) != 0 {
			config.CollectorURL = envURL
		} else {
			region := os.Getenv("AWS_REGION")
			if len(region) != 0 {
				config.CollectorURL = fmt.Sprintf("http://%s.tc.epsagon.com", region)
			} else {
				config.CollectorURL = "http://us-east-1.tc.epsagon.com"
			}
		}
		if config.Debug {
			log.Printf("EPSAGON DEBUG: setting collector url to %s\n", config.CollectorURL)
		}
	}
}

// CreateTracer will initiallize a global epsagon tracer
func CreateTracer(config *Config) {
	mutex.Lock()
	defer mutex.Unlock()
	if GlobalTracer != nil && !GlobalTracer.Stopped() {
		log.Println("The tracer is already created")
		return
	}
	if config == nil {
		config = &Config{}
	}
	fillConfigDefaults(config)
	GlobalTracer = &epsagonTracer{
		Config:         config,
		eventsPipe:     make(chan *protocol.Event),
		events:         make([]*protocol.Event, 0, 0),
		exceptionsPipe: make(chan *protocol.Exception),
		exceptions:     make([]*protocol.Exception, 0, 0),
		closeCmd:       make(chan struct{}),
		stopped:        make(chan struct{}),
		running:        make(chan struct{}),
	}
	if config.Debug {
		log.Println("EPSAGON DEBUG: Created a new tracer")
	}
	GlobalTracer.Start()
}

// AddException adds a tracing exception to the tracer
func (tracer *epsagonTracer) AddException(exception *protocol.Exception) {
	tracer.exceptionsPipe <- exception
}

// AddEvent adds an event to the tracer
func (tracer *epsagonTracer) AddEvent(event *protocol.Event) {
	if tracer.Config.Debug {
		log.Println("EPSAGON DEBUG: Adding event: ", event)
	}
	tracer.eventsPipe <- event
}

// AddEvent adds an event to the tracer
func AddEvent(event *protocol.Event) {
	if GlobalTracer == nil || GlobalTracer.Stopped() {
		// TODO
		log.Println("The tracer is not initialized!")
		return
	}
	GlobalTracer.AddEvent(event)
}

// AddException adds an exception to the tracer
func AddException(exception *protocol.Exception) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Epsagon: Failed to add exception")
		}
	}()
	if GlobalTracer == nil || GlobalTracer.Stopped() {
		// TODO
		log.Println("The tracer is not initialized!")
		return
	}
	GlobalTracer.AddException(exception)
}

// Stop stops the tracer running routine
func (tracer *epsagonTracer) Stop() {
	select {
	case <-tracer.stopped:
		return
	default:
		tracer.closeCmd <- struct{}{}
		<-tracer.stopped
	}
}

// StopTracer will close the tracer and send all the data to the collector
func StopTracer() {
	if GlobalTracer == nil || GlobalTracer.Stopped() {
		// TODO
		log.Println("The tracer is not initialized!")
		return
	}
	GlobalTracer.Stop()
}

// Run starts the runner background routine that will
// run until it
func (tracer *epsagonTracer) Run() {
	if tracer.Config.Debug {
		log.Println("EPSAGON DEBUG: tracer started running")
	}
	if tracer.Running() {
		return
	}
	close(tracer.running)
	defer func() { tracer.running = make(chan struct{}) }()
	defer close(tracer.stopped)

	for {
		select {
		case event := <-tracer.eventsPipe:
			tracer.events = append(tracer.events, event)
		case exception := <-tracer.exceptionsPipe:
			tracer.exceptions = append(tracer.exceptions, exception)
		case <-tracer.closeCmd:
			if tracer.Config.Debug {
				log.Println("EPSAGON DEBUG: tracer stops running, sending traces")
			}
			tracer.sendTraces()
			return
		}
	}
}

func (tracer *epsagonTracer) GetConfig() *Config {
	return tracer.Config
}

// GetGlobalTracerConfig returns the configuration of the global tracer
func GetGlobalTracerConfig() *Config {
	if GlobalTracer == nil || GlobalTracer.Stopped() {
		return &Config{}
	}
	return GlobalTracer.GetConfig()
}
