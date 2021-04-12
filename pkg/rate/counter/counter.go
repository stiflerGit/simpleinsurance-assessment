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

	// persistence
	savedAt              time.Time
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

// Must is equal to Nen but panics if there is some error
func Must(windowDuration time.Duration, resolution uint64, options ...Option) *Counter {
	wc, err := New(windowDuration, resolution, options...)
	if err != nil {
		panic(err)
	}
	return wc
}

func computePeriod(duration time.Duration, resolution uint64) time.Duration {
	return time.Duration(float64(duration) / float64(resolution))
}

// NewFromFile create a Counter starting from a state file
//
// the state file must be created by previously run of the Counter using
// the WithPersistence option
func NewFromFile(filePath string, options ...Option) (wc *Counter, err error) {
	c := &Counter{}

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

	if err = json.Unmarshal(bytes, c); err != nil {
		return nil, fmt.Errorf("unmashalling JSON: %v", err)
	}

	downDuration := time.Now().Sub(c.savedAt)
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
					c.savedAt = time.Now()
					if serr := c.saveState(); err != nil {
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

		c.counters = make([]int64, c.resolution)
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
}

func (c *Counter) saveState() error {
	f, err := os.Create(c.persistenceFilePath)
	if err != nil {
		return fmt.Errorf("creating file %s: %v", c.persistenceFilePath, err)
	}
	defer f.Close()

	bytes, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling json: %w", err)
	}

	if _, err = f.Write(bytes); err != nil {
		return fmt.Errorf("writing in file %s: %w", c.persistenceFilePath, err)
	}

	return nil
}

// Counter returns the number of increase received in the passed window
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

func (c *Counter) MarshalJSON() ([]byte, error) {
	c.m.Lock()
	defer c.m.Unlock()

	return json.Marshal(WindowCounterJSON{
		Duration:    c.windowDuration,
		Counter:     c.counter,
		PrevCounter: c.prevCounter,
		Resolution:  c.resolution,
		Counters:    c.counters,
		Head:        c.head,
		Tail:        c.tail,
		SavedAt:     c.savedAt,
	})
}

func (c *Counter) UnmarshalJSON(bytes []byte) error {
	c.m.Lock()
	defer c.m.Unlock()

	cJSON := WindowCounterJSON{}

	if err := json.Unmarshal(bytes, &cJSON); err != nil {
		return fmt.Errorf("unmarshalling json: %v", err)
	}

	c.windowDuration = cJSON.Duration
	c.counter = cJSON.Counter
	c.prevCounter = cJSON.PrevCounter
	c.resolution = cJSON.Resolution
	c.counters = cJSON.Counters
	c.head = cJSON.Head
	c.tail = cJSON.Tail
	c.savedAt = cJSON.SavedAt

	return nil
}