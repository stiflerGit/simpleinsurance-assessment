package rateCounter

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// TODO
type RateCounter struct {
	windows      time.Duration
	entries      []time.Time
	entriesMutex sync.Mutex
}

// TODO
func NewRateCounter(windowsDuration time.Duration, opts ...Option) *RateCounter {
	c := &RateCounter{
		windows: windowsDuration,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// TODO
func (c *RateCounter) RequestCounter() int {
	c.entriesMutex.Lock()
	defer c.entriesMutex.Unlock()

	windowsStart := time.Now().Add(-c.windows)

	i := 0
	for i = range c.entries {
		if c.entries[i].After(windowsStart) {
			break
		}
	}

	c.entries = c.entries[i:]
	return len(c.entries)
}

// TODO
func (c *RateCounter) Increase() {
	c.entriesMutex.Lock()
	defer c.entriesMutex.Unlock()

	c.entries = append(c.entries, time.Now())
}

func (c *RateCounter) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})

	m["windows"] = c.windows.String()

	entries := make([]string, 0, len(c.entries))
	for _, entry := range c.entries {
		entries = append(entries, entry.Format(time.RFC3339))
	}

	m["entries"] = entries

	return json.Marshal(m)
}

func (c *RateCounter) UnmarshalJSON(bytes []byte) error {
	m := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &m); err != nil {
		return err
	}

	{ // windows
		windows, ok := m["windows"]
		if !ok {
			return errors.New("\"windows\" key not present in JSON")
		}

		windowsStr, ok := windows.(string)
		if !ok {
			return errors.New("invalid \"windows\" value: expected string")
		}

		duration, err := time.ParseDuration(windowsStr)
		if err != nil {
			return fmt.Errorf("parsing windows duration %q: %v", windowsStr, err)
		}

		c.windows = duration
	}

	{ // entries
		entries, ok := m["entries"]
		if !ok {
			return errors.New("\"entries\" key not present in JSON")
		}

		entriesStrSlice, ok := entries.([]string)
		if !ok {
			return errors.New("invalid \"entriesStrSlice\" value: expected string slice")
		}

		c.entries = make([]time.Time, 0, len(entriesStrSlice))
		for i, entry := range entriesStrSlice {
			t, err := time.Parse(time.RFC3339, entry)
			if err != nil {
				return fmt.Errorf("parsing entry %s in position %d: %v", entry, i, err)
			}
			c.entries = append(c.entries, t)
		}
	}

	return nil
}
