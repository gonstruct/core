package cache

import "github.com/gonstruct/core/config"

type Option func(*Cache)

// WithConfig is required.
func WithConfig(cache config.Cache) Option {
	return func(c *Cache) {
		c.config = &cache
	}
}
