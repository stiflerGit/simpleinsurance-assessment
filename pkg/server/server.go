package server

import (
	"context"
	"encoding/json"
	"fmt"
	"local/simpleinsurance-assessment/pkg/rateCounter"
	"net/http"
	"time"
)

const (
	defaultWindowsDuration = 60 * time.Second
)

type Server struct {
	rateCounter *rateCounter.RateCounter
}

func New() (*Server, error) {
	rc, err := rateCounter.NewWindowCounter(defaultWindowsDuration)
	if err != nil {
		return nil, fmt.Errorf("creating new rateCounter: %v", err)
	}

	rc.Start(context.TODO())

	s := &Server{rateCounter: rc}

	return s, nil
}

func (s *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	s.rateCounter.Increase()
	requestCounter := s.rateCounter.Counter()

	m := map[string]interface{}{
		"counter": requestCounter,
	}

	bytes, err := json.Marshal(m)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = resp.Write(bytes); err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
	}

	resp.WriteHeader(http.StatusOK)
}
