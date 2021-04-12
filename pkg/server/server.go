package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/stiflerGit/simpleinsurance-assessment/pkg/rate/counter"
	"github.com/stiflerGit/simpleinsurance-assessment/pkg/rate/limiter"
)

const (
	defaultCounterWindowsDuration = 60 * time.Second
	defaultLimiterWindowsDuration = 20 * time.Second
	defaultCounterResolution      = 1000
	defaultPersistenceFilePath    = "windowCounterState.json"
	defaultSavePeriod             = 5 * time.Second
)

type Server struct {
	limiter  *limiter.Map
	counter  *counter.Counter
	logger   *log.Logger
	filePath string
	limit    int64
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
		return fmt.Errorf("building Value: %v", err)
	}
	s.counter = wc
	s.logger.Printf("window counter built\n")

	go func() {
		s.logger.Printf("starting window counter\n")
		if err := wc.Run(ctx); err != nil {
			panic(err)
		}
	}()

	if s.limit > 0 {
		l := limiter.NewMap(defaultLimiterWindowsDuration, s.limit)
		s.limiter = l
	}

	return nil
}

func (s Server) buildWindowCounter() (*counter.Counter, error) {

	rcOptions := []counter.Option{
		counter.WithPersistence(s.filePath, defaultSavePeriod),
	}

	if _, err := os.Stat(s.filePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stating file %s: %v", s.filePath, err)
		}
		return counter.New(defaultCounterWindowsDuration, defaultCounterResolution, rcOptions...)
	}

	return counter.NewFromFile(s.filePath, rcOptions...)
}

// ServeHTTP responds at each request with a counter of the total number
// of requests that it has received during the previous 60 seconds
func (s *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

	if s.limiter != nil {
		allowed, err := s.isClientAllowed(req)
		if err != nil {
			http.Error(resp, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if !allowed {
			http.Error(resp, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
	}

	response, err := s.Request()
	if err != nil {
		http.Error(resp, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(response)
	if err != nil {
		http.Error(resp, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if _, err = resp.Write(bytes); err != nil {
		http.Error(resp, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s Server) isClientAllowed(r *http.Request) (bool, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false, err // TODO: remove panic
	}

	lim := s.limiter.Get(ip)
	allowed := lim.IsAllowed()

	return allowed, nil
}

// Request execute the logic of the server i.e. return the number of requests in the last 60s
func (s *Server) Request() (Response, error) {
	return Response{
		Counter: s.counter.Increase(),
	}, nil
}

type Response struct {
	Counter int64 `json:"counter"`
}
