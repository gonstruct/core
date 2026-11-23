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

// WithCache is the cache queued jobs debounce in, asked for when the queue
// first runs: the app passes its own.
func WithCache(cache func() *cache.Cache) Option {
	return func(r *Queueing) {
		r.Cache = cache
	}
}

// WithConfig is how the queue runs: its driver, and how many workers the
// default queue and each group have. Queueing needs it.
func WithConfig(queue config.Queue) Option {
	return func(r *Queueing) {
		r.Config = &queue
	}
}
