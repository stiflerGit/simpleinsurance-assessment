package counter

import (
	"encoding/json"
	"fmt"
	"time"
)

type WindowCounterJSON struct {
	Duration    time.Duration `json:"windowDuration"`
	Counter     int64         `json:"counter"`
	PrevCounter int64         `json:"prev_counter"`
	Resolution  uint64        `json:"resolution"`
	Counters    []int64       `json:"counters"`
	Head        int           `json:"head"`
	Tail        int           `json:"tail"`
	At          time.Time     `json:"at"`
}

func (c *Counter) MarshalJSON() ([]byte, error) {
	c.m.Lock()
	defer c.m.Unlock()

	return json.Marshal(
		WindowCounterJSON{
			Duration:    c.windowDuration,
			Counter:     c.counter,
			PrevCounter: c.prevCounter,
			Resolution:  c.resolution,
			Counters:    c.counters,
			Head:        c.head,
			Tail:        c.tail,
			At:          c.at,
		},
	)
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
	c.at = cJSON.At

	return nil
}
