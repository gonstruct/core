package queueing

import "github.com/gonstruct/core/config"

type Option func(*Queueing)

func WithQueues(queues ...func(*Engine)) Option {
	return func(r *Queueing) {
		r.Queues = append(r.Queues, queues...)
	}
}

// WithConfig is required.
func WithConfig(queue config.Queue) Option {
	return func(r *Queueing) {
		r.Config = &queue
	}
}
