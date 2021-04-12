package limiter

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/stiflerGit/simpleinsurance-assessment/pkg/rate/counter"
)

const (
	defaultResolution = 1000
)

type Limiter struct {
	sync.Mutex
	c     *counter.Counter
	limit int64
}

// NewLimiter is the constructor of Limiter
func NewLimiter(duration time.Duration, limit int64) (*Limiter, error) {
	wc, err := counter.New(duration, defaultResolution)
	if err != nil {
		return nil, fmt.Errorf("creating counter: %v", err)
	}

	l := &Limiter{
		sync.Mutex{},
		wc,
		limit,
	}

	return l, nil
}

func NewLimiterFromJSON(bytes []byte) (*Limiter, error) {
	lJSON := LimiterJSON{}

	err := json.Unmarshal(bytes, &lJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling JSON: %v", err)
	}

	counterJSON, err := json.Marshal(lJSON.Counter)
	if err != nil {
		return nil, fmt.Errorf("marshalling counter: %v", err)
	}

	counter, err := counter.NewFromJSON(counterJSON)
	if err != nil {
		return nil, fmt.Errorf("creating new counterFromJSON")
	}

	return &Limiter{
		Mutex: sync.Mutex{},
		c:     counter,
		limit: lJSON.Limit,
	}, nil
}

// Must is same as NewLimiter but panics if there is some error
func Must(duration time.Duration, limit int64) *Limiter {
	l, err := NewLimiter(duration, limit)
	if err != nil {
		panic(err)
	}
	return l
}

// Start starts the limiter routine, it panics if it encounter some errors during the run
func (l *Limiter) Start(ctx context.Context) {
	go func() {
		if err := l.c.Run(ctx); err != nil {
			panic(err)
		}
	}()
}

// IsAllowed returns true if the number of request in the windows are under the limit
func (l *Limiter) IsAllowed() bool {
	l.Lock()
	defer l.Unlock()

	c := l.c.Value()

	if c >= l.limit {
		return false
	}

	l.c.Increase()
	return true
}
