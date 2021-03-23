package rateCounter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	defaultResolution = 1000
	minPeriod         = time.Millisecond
)

// RateCounter keeps information on the number of requests in the past time period
type RateCounter struct {
	duration    time.Duration
	counter     int64  // cumulative counter
	prevCounter int64  // value of the counter at previous tick
	resolution  uint64 // number of tick per window duration

	counters []int64 // counters per step, managed as a circular buffer
	head     int
	tail     int
}

// NewWindowCounter is the constructor of RateCounter
func NewWindowCounter(duration time.Duration, opts ...Option) (*RateCounter, error) {

	if duration < 0 {
		return nil, errors.New("negative duration")
	}

	c := &RateCounter{
		duration:   duration,
		resolution: defaultResolution,
	}

	for _, opt := range opts {
		opt(c)
	}

	period := computePeriod(c.duration, c.resolution)
	if period < minPeriod {
		return nil, fmt.Errorf("period less than minimum period: %v", period)
	}

	return c, nil
}

// Start starts the the RateCounter routine asynchronously
// to stop this routine just cancel the context
func (c *RateCounter) Start(ctx context.Context) {
	c.counters = make([]int64, c.resolution)

	period := computePeriod(c.duration, c.resolution)

	ticker := time.NewTicker(period)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return

			case <-ticker.C:
				c.tick()
			}
		}
	}()
}

func computePeriod(duration time.Duration, resolution uint64) time.Duration {
	return time.Duration(float64(duration) / float64(resolution))
}

func (c *RateCounter) tick() {
	counter := atomic.LoadInt64(&c.counter)

	// save actual diff
	c.counters[c.tail] = counter - c.prevCounter
	c.tail = (c.tail + 1) % len(c.counters)

	// tail = head means the window is full
	// and we can start to subtract oldest counters
	if c.tail == c.head {
		counter -= c.counters[c.head]
		c.head = (c.head + 1) % len(c.counters)
	}

	c.prevCounter = counter

	atomic.StoreInt64(&c.counter, counter)
}

// Counter returns the number of increase received in the passed window
func (c *RateCounter) Counter() int64 {
	return atomic.LoadInt64(&c.counter)
}

// Rate returns the number of increase per seconds
func (c *RateCounter) Rate() float64 {
	windowSeconds := float64(c.duration) / float64(time.Second)
	return float64(c.Counter()) / windowSeconds
}

// Increase increase the counter by one and returns the counter value
func (c *RateCounter) Increase() int64 {
	return atomic.AddInt64(&c.counter, 1)
}

func (c *RateCounter) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})

	m["duration"] = c.duration.String()
	m["counter"] = c.counter
	m["counters"] = c.counters
	m["resolution"] = c.resolution
	m["head"] = c.head
	m["tail"] = c.tail

	return json.Marshal(m)
}

func (c *RateCounter) UnmarshalJSON(bytes []byte) error {
	panic("implement me")
}
