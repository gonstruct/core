package queueing

type Option func(*Queueing)

func WithQueues(queues ...func(*Engine)) Option {
	return func(r *Queueing) {
		r.Queues = append(r.Queues, queues...)
	}
}
