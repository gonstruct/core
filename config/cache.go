package config

// cacheDriver is where cached values live.
type cacheDriver string

const (
	// MemoryCacheDriver keeps them in the process: gone on restart, and
	// not shared between processes.
	MemoryCacheDriver cacheDriver = "memory"
	// RedisCacheDriver keeps them in Redis, shared between processes.
	RedisCacheDriver cacheDriver = "redis"
)

// Cache is how the cache runs, handed to cache.WithConfig.
type Cache struct {
	Driver cacheDriver
	// Connection is the Redis connection the redis driver uses: the
	// default one when empty.
	Connection RedisConnectionType
}
