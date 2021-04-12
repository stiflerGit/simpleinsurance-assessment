package limiter

import (
	"encoding/json"
	"time"
)

type MapJSON struct {
	KeyToLimiter map[string]*Limiter `json:"key_to_limiter"`
	Duration     time.Duration       `json:"duration"`
	Limit        int64               `json:"limit"`
}

func (m *Map) MarshalJSON() ([]byte, error) {
	m.Lock()
	defer m.Unlock()

	mJSON := MapJSON{
		KeyToLimiter: m.keyToLimiter,
		Duration:     m.duration,
		Limit:        m.limit,
	}

	return json.Marshal(mJSON)
}

func (m *Map) UnmarshalJSON(bytes []byte) error {
	m.Lock()
	defer m.Unlock()

	mJSON := MapJSON{}
	err := json.Unmarshal(bytes, &mJSON)
	if err != nil {
		return err
	}

	m.duration = mJSON.Duration
	m.limit = mJSON.Limit
	m.keyToLimiter = mJSON.KeyToLimiter

	return nil
}
