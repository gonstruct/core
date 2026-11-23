package queueing

import (
	"context"
	"time"

	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/queueing/job"
	"github.com/gonstruct/core/queueing/resolvers"

	"github.com/rs/zerolog/log"

	"github.com/gonstruct/validation/env"
	"github.com/hibiken/asynq"
)

func NewEngine() (*Engine, error) {
	sharedCache := cache.New()

	return &Engine{
		EngineResolver: map[string]EngineResolver{
			"sync": &resolvers.Sync{Cache: sharedCache},
			"redis": &resolvers.Redis{
				Client: *asynq.NewClient(resolvers.RedisClient{}),
				Cache:  sharedCache,
			},
		}[env.String("QUEUE_DRIVER")],
		jobs:     make(map[string]job.Job),
		channels: make(map[string]int),
	}, nil
}

type Engine struct {
	EngineResolver

	jobs     map[string]job.Job
	channels map[string]int
}

type EngineResolver interface {
	Consume(jobs map[string]job.Job, channels map[string]int) error
	Dispatch(context.Context, job.Job, ...time.Duration) error
}

// Channel gives a queue workers of its own: up to concurrency of its jobs
// run at once, whatever the other queues are doing, as a Laravel Horizon
// supervisor. A job is on it by naming it in Queue(). A queue that is not a
// channel shares QUEUE_CONCURRENCY with the others that are not, by Priority.
func (e *Engine) Channel(queue string, concurrency int) {
	log.Info().Msgf("[queueing] registering channel %s with %d workers", queue, concurrency)

	e.channels[queue] = concurrency
}

func (e *Engine) Job(job job.Job) {
	log.Info().Msgf("[queueing] registering job %s", job.Name())

	e.jobs[job.Name()] = job
}
