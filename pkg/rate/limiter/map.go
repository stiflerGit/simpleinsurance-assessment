package limiter

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

type Map struct {
	sync.Mutex
	keyToLimiter         map[string]*Limiter
	duration             time.Duration
	limit                int64
	persistenceFilePath  string
	savePeriod           time.Duration
	isPersistenceEnabled bool
}

func NewMap(duration time.Duration, limit int64, options ...MapOption) *Map {
	m := &Map{
		Mutex:        sync.Mutex{},
		keyToLimiter: make(map[string]*Limiter),
		duration:     duration,
		limit:        limit,
	}

	for _, opt := range options {
		opt(m)
	}

	return m
}

func NewFromFile(filePath string, options ...MapOption) (*Map, error) {
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

func NewFromJSON(bytes []byte, options ...MapOption) (*Map, error) {
	mJSON := MapJSON{}
	if err := json.Unmarshal(bytes, &mJSON); err != nil {
		return nil, fmt.Errorf("unmarshalling json: %v", err)
	}

	keyToLimiter := make(map[string]*Limiter)

	// we must init each Limiter
	for key, limiter := range mJSON.KeyToLimiter {

		lJSON, err := json.Marshal(limiter)
		if err != nil {
			return nil, err
		}

		initLimiter, err := NewLimiterFromJSON(lJSON)
		if err != nil {
			return nil, err
		}

		keyToLimiter[key] = initLimiter
	}

	m := &Map{
		Mutex:        sync.Mutex{},
		keyToLimiter: keyToLimiter,
		duration:     mJSON.Duration,
		limit:        mJSON.Limit,
	}

	for _, opt := range options {
		opt(m)
	}

	return m, nil
}

func (m *Map) Run(ctx context.Context) error {
	if m.isPersistenceEnabled {

		ticker := time.NewTicker(m.savePeriod)

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return nil

			case <-ticker.C:
				if err := m.saveState(); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (m *Map) saveState() error {
	f, err := os.Create(m.persistenceFilePath)
	if err != nil {
		return fmt.Errorf("creating file %s: %v", m.persistenceFilePath, err)
	}
	defer f.Close()

	bytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshalling json: %w", err)
	}

	if _, err = f.Write(bytes); err != nil {
		return fmt.Errorf("writing in file %s: %w", m.persistenceFilePath, err)
	}

	return nil
}

func (m *Map) Get(key string) *Limiter {
	m.Lock()
	defer m.Unlock()

	l, ok := m.keyToLimiter[key]
	if !ok {
		l = Must(m.duration, m.limit)
		m.keyToLimiter[key] = l
	}

	return l
}
