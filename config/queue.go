package config

import "time"

// QueueDriver is how jobs run: sync where they are dispatched, redis on
// the worker.
type QueueDriver string

const (
	SynchronousQueueDriver QueueDriver = "sync"
	RedisQueueDriver       QueueDriver = "redis"
)

func (driver QueueDriver) Values() []QueueDriver {
	return []QueueDriver{
		SynchronousQueueDriver,
		RedisQueueDriver,
	}
}

func (driver QueueDriver) String() string {
	return string(driver)
}

// Queue is how the queue runs, handed to queueing.WithConfig: the driver,
// the workers of the default queue, and each group's.
type Queue struct {
	Driver QueueDriver
	// Concurrency is how many jobs of the default queue run at once: every
	// job registered outside a group.
	Concurrency int
	// CheckInterval is how long an idle default worker waits before it
	// looks for work again; 0 keeps asynq's second, with jitter.
	CheckInterval time.Duration
	// Groups is each queue group's workers of its own, by the name
	// engine.Group registers its jobs under. A group registered without one
	// stops the worker from starting.
	Groups      map[string]QueueGroup
	Connections struct {
		Redis RedisQueueConnection
	}
}

// QueueGroup is how a group's jobs run.
type QueueGroup struct {
	// Concurrency is how many of its jobs run at once, on workers of its
	// own: the other queues never take them.
	Concurrency int
	// CheckInterval is how long its idle workers wait before they look for
	// work again; 0 keeps asynq's second, with jitter. A job someone waits
	// on wants it short: on an idle queue it can sit up to 1.5s otherwise.
	CheckInterval time.Duration
}

type RedisQueueConnection struct {
	Driver     QueueDriver
	Connection RedisConnectionType
}
