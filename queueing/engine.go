package queueing

import (
	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/queueing/job"
	"github.com/gonstruct/core/queueing/resolvers"
	"time"

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

	jobs map[string]job.Job
}

type EngineResolver interface {
	Consume(map[string]job.Job) error
	Dispatch(job.Job, ...time.Duration) error
}

func (e *Engine) Job(job job.Job) {
	log.Info().Msgf("[queueing] registering job %s", job.Name())

	e.jobs[job.Name()] = job
}
