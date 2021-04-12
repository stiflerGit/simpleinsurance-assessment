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
	"path/filepath"
	"time"

	"github.com/stiflerGit/simpleinsurance-assessment/pkg/rate/counter"
	"github.com/stiflerGit/simpleinsurance-assessment/pkg/rate/limiter"
)

const (
	defaultCounterWindowsDuration     = 60 * time.Second
	defaultLimiterWindowsDuration     = 20 * time.Second
	defaultLimit                      = 15
	defaultCounterResolution          = 1000
	defaultPersistenceDir             = "persistence"
	defaultCounterPersistenceFileName = "windowCounterState.json"
	defaultLimiterPersistenceFileName = "limiter.json"
	defaultSavePeriod                 = time.Second
)

type Server struct {
	logger *log.Logger

	persistencePath string
	// counter
	counter *counter.Counter

	// limiter
	limiter *limiter.Map
	limit   int64
}

func New(opts ...Option) (*Server, error) {
	s := &Server{
		persistencePath: defaultPersistenceDir,
		limit:           defaultLimit,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Start start the server
func (s *Server) Start(ctx context.Context) error {

	if s.persistencePath != "" {
		if err := os.MkdirAll(s.persistencePath, os.ModePerm); err != nil {
			return err
		}
	}

	s.logger.Printf("building window counter\n")
	wc, err := s.buildWindowCounter()
	if err != nil {
		return fmt.Errorf("building counter: %v", err)
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
		s.logger.Printf("building limiter\n")
		limiter, err := s.buildLimiter()
		if err != nil {
			return fmt.Errorf("building LimiterMap: %v", err)
		}
		s.limiter = limiter
		s.logger.Printf("limiter built\n")

		go func() {
			s.logger.Printf("starting limiter\n")
			if err := limiter.Run(ctx); err != nil {
				panic(err)
			}
		}()
	}

	return nil
}

func (s Server) buildWindowCounter() (*counter.Counter, error) {

	counterFilePath := filepath.Join(s.persistencePath, defaultCounterPersistenceFileName)

	options := []counter.Option{
		counter.WithPersistence(counterFilePath, defaultSavePeriod),
	}

	if _, err := os.Stat(counterFilePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stating file %s: %v", counterFilePath, err)
		}
		return counter.New(defaultCounterWindowsDuration, defaultCounterResolution, options...)
	}

	return counter.NewFromFile(counterFilePath, options...)
}

func (s Server) buildLimiter() (*limiter.Map, error) {

	limiterFilePath := filepath.Join(s.persistencePath, defaultLimiterPersistenceFileName)

	options := []limiter.MapOption{
		limiter.WithPersistence(limiterFilePath, defaultSavePeriod),
	}

	if _, err := os.Stat(limiterFilePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stating file %s: %v", limiterFilePath, err)
		}
		return limiter.NewMap(defaultLimiterWindowsDuration, s.limit, options...), nil
	}

	return limiter.NewFromFile(limiterFilePath, options...)
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
		return false, err
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
