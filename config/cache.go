package config

// CacheDriver is where cached values live.
type CacheDriver string

const (
	// MemoryCacheDriver keeps them in the process: gone on restart, and
	// not shared between processes.
	MemoryCacheDriver CacheDriver = "memory"
	// RedisCacheDriver keeps them in Redis, shared between processes.
	RedisCacheDriver CacheDriver = "redis"
)

// Cache is how the cache runs, handed to cache.WithConfig.
type Cache struct {
	Driver CacheDriver
	// Connection is the Redis connection the redis driver uses: the
	// default one when empty.
	Connection RedisConnectionType
}
