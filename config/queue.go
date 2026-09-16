package config

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
	Driver      queueDriver
	Concurrency int
	Connections struct {
		Redis RedisQueueConnection
	}
}

type RedisQueueConnection struct {
	Driver     queueDriver
	Connection RedisConnectionType
}
