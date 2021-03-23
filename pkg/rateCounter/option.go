package rateCounter

type Option func(c *RateCounter)

func WithResolution(v uint64) Option {
	return func(c *RateCounter) {
		c.resolution = v
	}
}
