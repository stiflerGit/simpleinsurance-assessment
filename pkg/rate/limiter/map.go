package limiter

import (
	"sync"
	"time"
)

type Map struct {
	sync.Mutex
	keyToLimiter map[string]*Limiter
	duration     time.Duration
	limit        int64
}

func NewMap(duration time.Duration, limit int64) *Map {
	return &Map{
		Mutex:        sync.Mutex{},
		keyToLimiter: make(map[string]*Limiter),
		duration:     duration,
		limit:        limit,
	}
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
