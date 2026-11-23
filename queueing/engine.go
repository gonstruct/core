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
		jobs: make(map[string]job.Job),
	}, nil
}

type Engine struct {
	EngineResolver

	jobs        map[string]job.Job
	supervisors []resolvers.Supervisor
}

type EngineResolver interface {
	Consume(jobs map[string]job.Job, supervisors []resolvers.Supervisor) error
	Dispatch(context.Context, job.Job, ...time.Duration) error
}

// Supervisor adds a group of workers, as in Laravel Horizon: up to workers
// jobs at once, taken from its queues, in the order given. A
// job is on a queue by naming it in Queue(). The queues no supervisor reads
// share QUEUE_CONCURRENCY workers, by Priority.
//
//	engine.Supervisor("ci", 150, "ci")
//	engine.Supervisor("background", 100, "default")
func (e *Engine) Supervisor(name string, workers int, queues ...string) {
	log.Info().Msgf("[queueing] registering supervisor %s: %d workers for %v", name, workers, queues)

	e.supervisors = append(e.supervisors, resolvers.Supervisor{Name: name, Workers: workers, Queues: queues})
}

func (e *Engine) Job(job job.Job) {
	log.Info().Msgf("[queueing] registering job %s", job.Name())

	e.jobs[job.Name()] = job
}
