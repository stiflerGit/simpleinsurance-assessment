package windowCounter

import "time"

// TODO
type Option func(c *WindowCounter)

// TODO
func WithPersistence(filePath string, savePeriod time.Duration) Option {
	return func(c *WindowCounter) {
		c.persistenceFilePath = filePath
		c.savePeriod = savePeriod
	}
}
