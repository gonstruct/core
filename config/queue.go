package config

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
}

type RedisQueueConnection struct {
	Driver     QueueDriver
	Connection RedisConnectionType
}
