package cache

import "github.com/gonstruct/core/config"

type Option func(*Cache)

// WithConfig is how the cache runs: its driver. A cache used without one
// fails.
func WithConfig(cache config.Cache) Option {
	return func(c *Cache) {
		c.config = &cache
	}
}
