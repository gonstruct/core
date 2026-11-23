package config

import "time"

type queueDriver string

const (
	SynchronousQueueDriver queueDriver = "sync"
	RedisQueueDriver       queueDriver = "redis"
)

func (driver queueDriver) Values() []queueDriver {
	return []queueDriver{
		SynchronousQueueDriver,
		RedisQueueDriver,
	}
}

func (driver queueDriver) String() string {
	return string(driver)
}

type Queue struct {
	Driver queueDriver
	// Concurrency and CheckInterval are for the workers that run every job
	// outside a group.
	Concurrency   int
	CheckInterval time.Duration // 0 uses asynq's default of 1s
	// Groups gives each engine.Group its own workers, by group name.
	Groups      map[string]QueueGroup
	Connections struct {
		Redis RedisQueueConnection
	}
}

type QueueGroup struct {
	Concurrency   int
	CheckInterval time.Duration // 0 uses asynq's default of 1s
}

type RedisQueueConnection struct {
	Driver     queueDriver
	Connection RedisConnectionType
}
