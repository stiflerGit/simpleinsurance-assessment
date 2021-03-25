package windowCounter

import (
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
	SavedAt     time.Time     `json:"saved_at"`
}
