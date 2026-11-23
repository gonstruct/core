package config

type Redis struct {
	Options     any
	Connections map[RedisConnectionType]RedisConnection
}

type RedisConnection struct {
	Mode             redisMode // default single
	Addresses        []string  // the servers, or the sentinels; Host and Port also work for one server
	Host             string
	Port             string
	MasterName       string // sentinel mode only
	Username         string
	Password         string
	SentinelUsername string
	SentinelPassword string
	Database         int // 0 to 15; must be 0 in cluster mode
	TLS              bool
}

type redisMode string

const (
	SingleRedisMode   redisMode = "single"
	SentinelRedisMode redisMode = "sentinel"
	ClusterRedisMode  redisMode = "cluster"
)

func (mode redisMode) Values() []redisMode {
	return []redisMode{SingleRedisMode, SentinelRedisMode, ClusterRedisMode}
}

func (mode redisMode) String() string {
	return string(mode)
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
