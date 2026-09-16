package config

type Redis struct {
	Options     any
	Connections map[RedisConnectionType]RedisConnection
}

type RedisConnection struct {
	Host     string
	Port     string
	Username string
	Password string
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
