package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	filePath      string
}

func New(opts ...Option) (*Server, error) {
	s := &Server{
		filePath: defaultPersistenceFilePath,
	}

	for _, opt := range opts {
		opt(s)
	}

	wc, err := s.buildWindowCounter()
	if err != nil {
		return nil, fmt.Errorf("building WindowCounter: %v", err)
	}
	s.windowCounter = wc

	wc.Start(context.TODO())

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

func (s *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	type responseJSON struct {
		Counter int64 `json:"counter"`
	}

	respJSON := responseJSON{Counter: s.windowCounter.Increase()}

	bytes, err := json.Marshal(respJSON)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = resp.Write(bytes); err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
	}

	resp.WriteHeader(http.StatusOK)
}
