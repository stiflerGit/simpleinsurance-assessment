package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/stiflerGit/simpleinsurance-assessment/pkg/windowCounter"
)

const (
	defaultWindowsDuration     = 60 * time.Second
	defaultCounterResolution   = 1000
	defaultPersistenceFilePath = "windowCounterState.json"
	defaultSavePeriod          = 5 * time.Second
)

type Server struct {
	windowCounter *windowCounter.WindowCounter
	logger        *log.Logger
	filePath      string
	err           error
}

func New(opts ...Option) (*Server, error) {
	s := &Server{
		filePath: defaultPersistenceFilePath,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Start start the server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Printf("building window counter\n")
	wc, err := s.buildWindowCounter()
	if err != nil {
		return fmt.Errorf("building WindowCounter: %v", err)
	}
	s.windowCounter = wc
	s.logger.Printf("window counter built\n")

	go func() {
		s.logger.Printf("starting window counter\n")
		if err := wc.Run(ctx); err != nil {
			panic(err)
		}
	}()

	return nil
}

func (s Server) buildWindowCounter() (*windowCounter.WindowCounter, error) {

	rcOptions := []windowCounter.Option{
		windowCounter.WithPersistence(s.filePath, defaultSavePeriod),
	}

	if _, err := os.Stat(s.filePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stating file %s: %v", s.filePath, err)
		}
		return windowCounter.New(defaultWindowsDuration, defaultCounterResolution, rcOptions...)
	}

	return windowCounter.NewFromFile(s.filePath, rcOptions...)
}

// ServeHTTP responds at each request with a counter of the total number
// of requests that it has received during the previous 60 seconds
func (s *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	response, err := s.Request()
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(response)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = resp.Write(bytes); err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
	}
}

// Request execute the logic of the server i.e. return the number of requests in the last 60s
func (s *Server) Request() (Response, error) {
	return Response{
		Counter: s.windowCounter.Increase(),
	}, nil
}

type Response struct {
	Counter int64 `json:"counter"`
}
