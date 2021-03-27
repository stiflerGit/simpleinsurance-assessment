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

	s.logger.Printf("building window counter\n")
	wc, err := s.buildWindowCounter()
	if err != nil {
		return nil, fmt.Errorf("building WindowCounter: %v", err)
	}
	s.windowCounter = wc
	s.logger.Printf("window counter built\n")

	go func() {
		for {
			s.logger.Printf("starting window counter\n")
			if err := wc.Run(context.TODO()); err != nil {
				s.logger.Printf("[WARN] window counter error: %v\n", err)
				s.logger.Printf("trying to restart window counter\n")

				s.logger.Printf("removing old persistence file\n")
				if err = os.Remove(s.filePath); err != nil {
					s.logger.Printf("removing persistence file in %s: %v\n", s.filePath, err)
				}
				s.logger.Printf("old persistence file removed\n")

				s.logger.Printf("building window counter\n")
				wc, err = s.buildWindowCounter()
				if err != nil {
					panic(fmt.Errorf("building WindowCounter: %v", err))
				}
				s.logger.Printf("window counter built\n")
				s.windowCounter = wc
			}
		}
	}()

	return s, nil
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

	resp.WriteHeader(http.StatusOK)
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
