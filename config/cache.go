package config

type cacheDriver string

const (
	MemoryCacheDriver cacheDriver = "memory"
	RedisCacheDriver  cacheDriver = "redis"
)

func (driver cacheDriver) Values() []cacheDriver {
	return []cacheDriver{MemoryCacheDriver, RedisCacheDriver}
}

func (driver cacheDriver) String() string {
	return string(driver)
}

type Cache struct {
	Driver     cacheDriver
	Connection RedisConnectionType // the redis driver's connection; default when empty
}
