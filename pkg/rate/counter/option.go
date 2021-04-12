package counter

import "time"

type Option func(c *Counter)

// WithPersistence set the Counter to save its state in a file
//
// filePath is the path where the state file will be saved, the state
// is stored each savePeriod, overwriting the previous saved state
func WithPersistence(filePath string, savePeriod time.Duration) Option {
	return func(c *Counter) {
		c.persistenceFilePath = filePath
		c.savePeriod = savePeriod
		c.isPersistenceEnabled = true
	}
}
