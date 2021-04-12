package counter

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

const (
	minPeriod = time.Millisecond
)

// Counter keeps information on the number of requests in the past time period
type Counter struct {
	m              sync.Mutex
	windowDuration time.Duration
	counter        int64  // cumulative counter
	prevCounter    int64  // value of the counter at previous tick
	resolution     uint64 // number of tick per window windowDuration

	counters []int64 // counters per tick, managed as a circular buffer
	head     int
	tail     int
	at       time.Time

	// persistence
	persistenceFilePath  string
	savePeriod           time.Duration
	isPersistenceEnabled bool

	stop chan struct{}
}

// New is the constructor of Counter
func New(windowDuration time.Duration, resolution uint64, options ...Option) (*Counter, error) {
	c := &Counter{
		windowDuration: windowDuration,
		resolution:     resolution,
		counters:       make([]int64, resolution),
	}

	tickPeriod := computePeriod(c.windowDuration, c.resolution)
	if tickPeriod < minPeriod {
		return nil, fmt.Errorf("tickPeriod less than minimum tickPeriod: %v", tickPeriod)
	}

	for _, opt := range options {
		opt(c)
	}

	return c, nil
}

// Must is equal to New but panics if there is some error
func Must(windowDuration time.Duration, resolution uint64, options ...Option) *Counter {
	c, err := New(windowDuration, resolution, options...)
	if err != nil {
		panic(err)
	}
	return c
}

func computePeriod(duration time.Duration, resolution uint64) time.Duration {
	return time.Duration(float64(duration) / float64(resolution))
}

// NewFromFile create a Counter starting from a state file
//
// the state file must be created by previously run of the Counter using
// the WithPersistence option
func NewFromFile(filePath string, options ...Option) (wc *Counter, err error) {

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %v", filePath, err)
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %v", filePath, err)
	}

	return NewFromJSON(bytes, options...)
}

// NewFromJSON create a Counter starting from a JSON input
func NewFromJSON(bytes []byte, options ...Option) (*Counter, error) {
	c := &Counter{}

	if err := json.Unmarshal(bytes, c); err != nil {
		return nil, fmt.Errorf("unmashalling JSON: %v", err)
	}

	downDuration := time.Now().Sub(c.at)
	tickPeriod := computePeriod(c.windowDuration, c.resolution)
	missingTicks := int(float64(downDuration) / float64(tickPeriod))

	// simulate missing ticks
	for i := 0; i < missingTicks; i++ {
		c.tick()
	}

	for _, opt := range options {
		opt(c)
	}

	return c, nil
}

// Run runs the the Counter routine
//
// to stop this routine just cancel the context
func (c *Counter) Run(ctx context.Context) error {
	var (
		err error
		wg  sync.WaitGroup
	)

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	if c.isPersistenceEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ticker := time.NewTicker(c.savePeriod)

			for {
				select {
				case <-ctx.Done():
					ticker.Stop()
					return

				case <-ticker.C:
					if serr := c.saveState(); serr != nil {
						cancelFunc()
						err = serr
						return
					}
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		period := computePeriod(c.windowDuration, c.resolution)
		ticker := time.NewTicker(period)

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return

			case <-c.stop:
				ticker.Stop()
				return

			case <-ticker.C:
				c.tick()
			}
		}
	}()

	wg.Wait()

	return err
}

func (c *Counter) tick() {
	c.m.Lock()
	defer c.m.Unlock()

	// save actual diff
	c.counters[c.head] = c.counter - c.prevCounter
	c.head = (c.head + 1) % len(c.counters)

	// tail = head means the window is full
	// and we can start to subtract oldest counters
	if c.tail == c.head {
		c.counter -= c.counters[c.tail]
		c.tail = (c.tail + 1) % len(c.counters)
	}

	c.prevCounter = c.counter

	c.at = time.Now()
}

func (c *Counter) saveState() (err error) {
	// mutex lock is in MarshalJSON

	f, cerr := os.Create(c.persistenceFilePath)
	if cerr != nil {
		return fmt.Errorf("creating file %s: %v", c.persistenceFilePath, cerr)
	}
	defer func() {
		if cerr = f.Close(); err == nil {
			err = cerr
		}
	}()

	bytes, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling json: %w", err)
	}

	if _, err = f.Write(bytes); err != nil {
		return fmt.Errorf("writing in file %s: %w", c.persistenceFilePath, err)
	}

	return nil
}

// Value returns the number of increase received in the passed window
func (c *Counter) Value() int64 {
	c.m.Lock()
	defer c.m.Unlock()

	return c.counter
}

// Rate returns the number of increase per seconds
func (c *Counter) Rate() float64 {
	windowSeconds := float64(c.windowDuration) / float64(time.Second)
	return float64(c.Value()) / windowSeconds
}

// Increase increase the counter by one and returns the counter value
func (c *Counter) Increase() int64 {
	c.m.Lock()
	defer c.m.Unlock()

	c.counter++
	return c.counter
}
