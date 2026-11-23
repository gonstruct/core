package redis_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gonstruct/core/config"
	redisconnection "github.com/gonstruct/core/redis"
)

func configure(connections map[config.RedisConnectionType]config.RedisConnection) {
	redisconnection.Configure(config.Redis{Connections: connections})
}

func TestAConnectionNeedsAConfig(t *testing.T) {
	// Before any Configure in this process: the first test to run.
	if _, err := redisconnection.Load("default"); err == nil || !strings.Contains(err.Error(), "no config") {
		t.Fatalf("expected no config, got %v", err)
	}
}

func TestLoadUsesTheDefaultConnection(t *testing.T) {
	configure(map[config.RedisConnectionType]config.RedisConnection{
		config.DefaultRedisConnection: {Host: "shared-redis", Port: "6379"},
	})

	configuration, err := redisconnection.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Mode != redisconnection.ModeSingle || len(configuration.Addresses) != 1 || configuration.Addresses[0] != "shared-redis:6379" {
		t.Fatalf("unexpected default Redis configuration: %#v", configuration)
	}
}

func TestLoadBuildsANamedSentinelConnection(t *testing.T) {
	configure(map[config.RedisConnectionType]config.RedisConnection{
		"queue": {
			Mode:             config.SentinelRedisMode,
			Addresses:        []string{"sentinel-1:26379", "sentinel-2:26379", "sentinel-3:26379"},
			MasterName:       "example-queue",
			SentinelUsername: "sentinel-user",
			SentinelPassword: "sentinel-secret",
		},
	})

	configuration, err := redisconnection.Load("queue")
	if err != nil {
		t.Fatal(err)
	}
	options := configuration.Options()
	if len(options.Addrs) != 3 || options.MasterName != "example-queue" || options.SentinelUsername != "sentinel-user" ||
		options.SentinelPassword != "sentinel-secret" {
		t.Fatalf("unexpected Sentinel configuration: %#v", configuration)
	}
}

func TestLoadAllowsACluster(t *testing.T) {
	configure(map[config.RedisConnectionType]config.RedisConnection{
		"cache": {Mode: config.ClusterRedisMode, Addresses: []string{"cache-1:6379", "cache-2:6379", "cache-3:6379"}},
	})

	configuration, err := redisconnection.Load("cache")
	if err != nil || !configuration.Options().IsClusterMode || len(configuration.Options().Addrs) != 3 {
		t.Fatalf("unexpected cluster configuration: %#v (%v)", configuration, err)
	}
}

func TestLoadRefuses(t *testing.T) {
	configure(map[config.RedisConnectionType]config.RedisConnection{
		"no-address": {},
		"no-master":  {Mode: config.SentinelRedisMode, Addresses: []string{"sentinel-1:26379"}},
		"cluster-db": {Mode: config.ClusterRedisMode, Addresses: []string{"cache-1:6379"}, Database: 2},
		"bad-mode":   {Mode: "ring", Addresses: []string{"a:1"}},
		"bad-db":     {Host: "a", Port: "1", Database: 16},
	})

	for _, name := range []string{"no-address", "no-master", "cluster-db", "bad-mode", "bad-db", "missing"} {
		if _, err := redisconnection.Load(name); err == nil {
			t.Errorf("expected %s refused", name)
		}
	}
}

func TestConfiguredClientConnectsToRedis(t *testing.T) {
	if os.Getenv("REDIS_TOPOLOGY_INTEGRATION") != "1" {
		t.Skip("set REDIS_TOPOLOGY_INTEGRATION=1, REDIS_HOST and REDIS_PORT to run the Redis topology integration test")
	}
	configure(map[config.RedisConnectionType]config.RedisConnection{
		config.DefaultRedisConnection: {Host: os.Getenv("REDIS_HOST"), Port: os.Getenv("REDIS_PORT")},
	})

	client, err := redisconnection.NewClient(redisconnection.DefaultConnection)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	pingContext, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := client.Ping(pingContext).Err(); err != nil {
		t.Fatalf("configured Redis client could not connect: %v", err)
	}
}
