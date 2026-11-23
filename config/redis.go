package config

// Redis is the Redis connections the app uses, by name, handed to
// app.WithRedis: the queue and the cache name the one they use.
type Redis struct {
	Options     any
	Connections map[RedisConnectionType]RedisConnection
}

// RedisConnection is one Redis: a single server, a Sentinel-watched one, or
// a cluster.
type RedisConnection struct {
	// Mode is single (the default when empty), sentinel or cluster.
	Mode string
	// Addresses is its servers, or its sentinels; or Host and Port for one.
	Addresses []string
	Host      string
	Port      string
	// MasterName is the master the sentinels watch, in sentinel mode.
	MasterName       string
	Username         string
	Password         string
	SentinelUsername string
	SentinelPassword string
	// Database is 0 to 15; always 0 in cluster mode.
	Database int
	TLS      bool
}

type RedisConnectionType string

const (
	DefaultRedisConnection RedisConnectionType = "default"
)

func (r RedisConnectionType) Values() (values []RedisConnectionType) {
	values = append(values, []RedisConnectionType{DefaultRedisConnection}...)
	return
}

func (r RedisConnectionType) String() string {
	return string(r)
}
