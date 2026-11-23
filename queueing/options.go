package queueing

import "github.com/gonstruct/core/config"

type Option func(*Queueing)

func WithQueues(queues ...func(*Engine)) Option {
	return func(r *Queueing) {
		r.Queues = append(r.Queues, queues...)
	}
}

// WithConfig is how the queue runs: its driver, and how many workers the
// default queue and each group have. Queueing needs it.
func WithConfig(queue config.Queue) Option {
	return func(r *Queueing) {
		r.Config = &queue
	}
}
