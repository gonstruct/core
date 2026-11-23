package redis

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync/atomic"

	"github.com/gonstruct/core/config"

	goredis "github.com/redis/go-redis/v9"
)

const (
	DefaultConnection = "default"

	ModeSingle   = string(config.SingleRedisMode)
	ModeSentinel = string(config.SentinelRedisMode)
	ModeCluster  = string(config.ClusterRedisMode)
)

type Configuration struct {
	Mode             string
	Addresses        []string
	MasterName       string
	Username         string
	Password         string
	SentinelUsername string
	SentinelPassword string
	Database         int
	TLS              bool
}

// configured is the app's connections, by name.
var configured atomic.Pointer[config.Redis]

// Configure registers the app's Redis connections. app.WithRedis calls it.
func Configure(redis config.Redis) {
	configured.Store(&redis)
}

// Load is the connection by that name, as configured: the default one when
// empty.
func Load(connection string) (Configuration, error) {
	redis := configured.Load()
	if redis == nil {
		return Configuration{}, errors.New("[redis] no config: pass app.WithRedis")
	}
	if strings.TrimSpace(connection) == "" {
		connection = DefaultConnection
	}
	settings, found := redis.Connections[config.RedisConnectionType(connection)]
	if !found {
		return Configuration{}, fmt.Errorf("[redis] no connection %q in the config", connection)
	}

	mode, addresses := ModeSingle, settings.Addresses
	if settings.Mode != "" {
		mode = settings.Mode.String()
	}
	if len(addresses) == 0 && settings.Host != "" && settings.Port != "" {
		addresses = []string{net.JoinHostPort(settings.Host, settings.Port)}
	}

	configuration := Configuration{
		Mode:             mode,
		Addresses:        addresses,
		MasterName:       settings.MasterName,
		Username:         settings.Username,
		Password:         settings.Password,
		SentinelUsername: settings.SentinelUsername,
		SentinelPassword: settings.SentinelPassword,
		Database:         settings.Database,
		TLS:              settings.TLS,
	}
	if err := configuration.validate(connection); err != nil {
		return Configuration{}, err
	}

	return configuration, nil
}

func NewClient(connection string) (goredis.UniversalClient, error) {
	configuration, err := Load(connection)
	if err != nil {
		return nil, err
	}

	return goredis.NewUniversalClient(configuration.Options()), nil
}

func (configuration Configuration) Options() *goredis.UniversalOptions {
	options := &goredis.UniversalOptions{
		Addrs:            configuration.Addresses,
		Username:         configuration.Username,
		Password:         configuration.Password,
		SentinelUsername: configuration.SentinelUsername,
		SentinelPassword: configuration.SentinelPassword,
		DB:               configuration.Database,
	}

	switch configuration.Mode {
	case ModeSentinel:
		options.MasterName = configuration.MasterName
	case ModeCluster:
		options.IsClusterMode = true
	}

	if configuration.TLS {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	return options
}

func (configuration Configuration) validate(connection string) error {
	if configuration.Database < 0 || configuration.Database > 15 {
		return fmt.Errorf("[redis] connection %s has Database %d: 0 to 15", connection, configuration.Database)
	}

	switch configuration.Mode {
	case ModeSingle:
		if len(configuration.Addresses) != 1 {
			return fmt.Errorf("[redis] connection %s needs one address in single mode", connection)
		}
	case ModeSentinel:
		if configuration.MasterName == "" {
			return fmt.Errorf("[redis] connection %s needs a MasterName in sentinel mode", connection)
		}
	case ModeCluster:
		if configuration.Database != 0 {
			return fmt.Errorf("[redis] connection %s must use Database 0 in cluster mode", connection)
		}
	default:
		return fmt.Errorf("[redis] connection %s has Mode %q: single, sentinel or cluster", connection, configuration.Mode)
	}

	return nil
}
