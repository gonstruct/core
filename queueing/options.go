package queueing

import (
	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/config"
)

type Option func(*Queueing)

func WithQueues(queues ...func(*Engine)) Option {
	return func(r *Queueing) {
		r.Queues = append(r.Queues, queues...)
	}
}

// WithCache sets the cache debounced jobs use. It is called when the queue
// first runs.
func WithCache(cache func() *cache.Cache) Option {
	return func(r *Queueing) {
		r.Cache = cache
	}
}

// WithConfig is required.
func WithConfig(queue config.Queue) Option {
	return func(r *Queueing) {
		r.Config = &queue
	}
}
