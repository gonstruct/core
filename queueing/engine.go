package queueing

import (
	"context"
	"fmt"
	"time"

	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/config"
	"github.com/gonstruct/core/queueing/job"
	"github.com/gonstruct/core/queueing/resolvers"

	"github.com/rs/zerolog/log"

	"github.com/hibiken/asynq"
)

// NewEngine is the queue running as configured, debouncing jobs in the
// cache.
func NewEngine(queue config.Queue, sharedCache *cache.Cache) (*Engine, error) {
	var resolver EngineResolver
	switch queue.Driver {
	case config.SynchronousQueueDriver:
		resolver = &resolvers.Sync{Cache: sharedCache}
	case config.RedisQueueDriver:
		connection := resolvers.RedisClient{Connection: queue.Connections.Redis.Connection.String()}
		resolver = &resolvers.Redis{Client: *asynq.NewClient(connection), Connection: connection, Cache: sharedCache, Config: queue}
	default:
		return nil, fmt.Errorf("[queueing] unknown queue driver %q", queue.Driver)
	}

	return &Engine{EngineResolver: resolver, jobs: make(map[string]job.Job), groups: make(map[string]string)}, nil
}

type Engine struct {
	EngineResolver

	jobs map[string]job.Job
	// groups is the group each job registered in one is in, by job name.
	groups map[string]string
}

type EngineResolver interface {
	// Consume runs the jobs; groups names the group of each job in one.
	Consume(jobs map[string]job.Job, groups map[string]string) error
	// Dispatch puts a job on a queue: its group's, or "" for the job's own.
	Dispatch(ctx context.Context, job job.Job, queue string, delay ...time.Duration) error
}

func (e *Engine) Job(job job.Job) {
	log.Info().Msgf("[queueing] registering job %s", job.Name())

	e.jobs[job.Name()] = job
}

// Group registers jobs on a queue of their own, with workers of their own,
// as many as the config's Groups says for its name:
//
//	engine.Group("ci", func(group *queueing.Group) {
//		group.Job(new(LaunchRunnerJob))
//	})
//
// A job in a group goes on the group's queue, whatever its Queue() says,
// and only the group's workers run it, so no other job can hold it up. Jobs
// outside a group share the default workers; their Queue() and Priority()
// only weight which of them those take next.
func (e *Engine) Group(name string, register func(*Group)) {
	register(&Group{name: name, engine: e})
}

// QueueOf is the queue a job goes on: its group's, or "" when it is in none
// and says for itself.
func (e *Engine) QueueOf(job job.Job) string {
	return e.groups[job.Name()]
}

// Group is the jobs registering in one.
type Group struct {
	name   string
	engine *Engine
}

func (g *Group) Job(job job.Job) {
	if named, ok := job.(interface{ Queue() string }); ok && named.Queue() != g.name {
		log.Warn().Msgf("[queueing] job %s says queue %s but is in group %s: it goes on %s", job.Name(), named.Queue(), g.name, g.name)
	}
	log.Info().Msgf("[queueing] registering job %s in group %s", job.Name(), g.name)

	g.engine.jobs[job.Name()] = job
	g.engine.groups[job.Name()] = g.name
}
