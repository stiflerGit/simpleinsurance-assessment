package limiter

import "time"

type MapOption func(m *Map)

// WithPersistence set the Counter to save its state in a file
//
// filePath is the path where the state file will be saved, the state
// is stored each savePeriod, overwriting the previous saved state
func WithPersistence(filePath string, savePeriod time.Duration) MapOption {
	return func(m *Map) {
		m.persistenceFilePath = filePath
		m.savePeriod = savePeriod
		m.isPersistenceEnabled = true
	}
}
