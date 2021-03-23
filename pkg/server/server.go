package server

import (
	"encoding/json"
	"local/simpleinsurance-assessment/pkg/rateCounter"
	"net/http"
	"time"
)

const (
	defaultWindowsDuration = 60 * time.Second
)

type Server struct {
	rateCounter rateCounter.RateCounter
}

func New() *Server {
	s := &Server{rateCounter: *rateCounter.NewRateCounter(defaultWindowsDuration)}
	return s
}

func (s *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	s.rateCounter.Increase()
	requestCounter := s.rateCounter.RequestCounter()

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
