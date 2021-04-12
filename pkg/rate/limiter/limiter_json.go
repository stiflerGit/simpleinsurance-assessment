package limiter

import (
	"encoding/json"

	"github.com/stiflerGit/simpleinsurance-assessment/pkg/rate/counter"
)

type LimiterJSON struct {
	Counter *counter.Counter `json:"counter"`
	Limit   int64            `json:"limit"`
}

func (l *Limiter) MarshalJSON() ([]byte, error) {
	l.Lock()
	defer l.Unlock()

	lJSON := LimiterJSON{
		Counter: l.c,
		Limit:   l.limit,
	}

	return json.Marshal(lJSON)
}

func (l *Limiter) UnmarshalJSON(bytes []byte) error {
	l.Lock()
	defer l.Unlock()

	lJSON := LimiterJSON{}
	if err := json.Unmarshal(bytes, &lJSON); err != nil {
		return err
	}

	l.limit = lJSON.Limit
	l.c = lJSON.Counter

	return nil
}
