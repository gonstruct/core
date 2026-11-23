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

func NewEngine(queue config.Queue) (*Engine, error) {
	sharedCache := cache.New()

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

	return &Engine{EngineResolver: resolver, jobs: map[string]job.Job{}, groups: map[string]string{}}, nil
}

type Engine struct {
	EngineResolver

	jobs   map[string]job.Job
	groups map[string]string // job name -> group name
}

type EngineResolver interface {
	Consume(jobs map[string]job.Job, groups map[string]string) error
	Dispatch(ctx context.Context, job job.Job, queue string, delay ...time.Duration) error
}

func (e *Engine) Job(job job.Job) {
	log.Info().Msgf("[queueing] registering job %s", job.Name())

	e.jobs[job.Name()] = job
}

// Group registers jobs that run on workers of their own, so jobs outside
// the group can never hold them up. config.Queue.Groups says how many.
//
//	engine.Group("ci", func(group *queueing.Group) {
//		group.Job(new(LaunchRunnerJob))
//	})
//
// Jobs outside a group share the default workers, which pick between the
// jobs' Queue() by Priority().
func (e *Engine) Group(name string, register func(*Group)) {
	register(&Group{name: name, engine: e})
}

// QueueOf returns the queue a job is dispatched to: its group's, the one
// it names with Queue(), or the default queue.
func (e *Engine) QueueOf(current job.Job) string {
	if group, ok := e.groups[current.Name()]; ok {
		return group
	}
	if named, ok := current.(job.JobWithQueue); ok {
		return named.Queue()
	}

	return resolvers.DefaultQueue
}

type Group struct {
	name   string
	engine *Engine
}

func (g *Group) Job(current job.Job) {
	if named, ok := current.(job.JobWithQueue); ok && named.Queue() != g.name {
		log.Warn().Msgf("[queueing] job %s is in group %s, so its Queue() %s is ignored", current.Name(), g.name, named.Queue())
	}
	log.Info().Msgf("[queueing] registering job %s in group %s", current.Name(), g.name)

	g.engine.jobs[current.Name()] = current
	g.engine.groups[current.Name()] = g.name
}
