package redis_test

import (
	"context"
	"os"
	"testing"
	"time"

	redisconnection "github.com/gonstruct/core/redis"
)

func TestLoadUsesDefaultConnectionSettings(t *testing.T) {
	configureSharedRedis(t)

	configuration, err := redisconnection.Load(redisconnection.DefaultConnection)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Mode != redisconnection.ModeSingle ||
		len(configuration.Addresses) != 1 || configuration.Addresses[0] != "shared-redis:6379" {
		t.Fatalf("unexpected default Redis configuration: %#v", configuration)
	}
}

func TestLoadBuildsNamedSentinelConnection(t *testing.T) {
	configureSharedRedis(t)
	t.Setenv("REDIS_QUEUE_MODE", "sentinel")
	t.Setenv("REDIS_QUEUE_ADDRESSES", "sentinel-1:26379, sentinel-2:26379, sentinel-3:26379")
	t.Setenv("REDIS_QUEUE_MASTER_NAME", "example-queue")
	t.Setenv("REDIS_QUEUE_SENTINEL_USERNAME", "sentinel-user")
	t.Setenv("REDIS_QUEUE_SENTINEL_PASSWORD", "sentinel-secret")

	configuration, err := redisconnection.Load("queue")
	if err != nil {
		t.Fatal(err)
	}
	options := configuration.Options()
	if configuration.Mode != redisconnection.ModeSentinel || len(options.Addrs) != 3 ||
		options.MasterName != "example-queue" || options.SentinelUsername != "sentinel-user" ||
		options.SentinelPassword != "sentinel-secret" {
		t.Fatalf("unexpected Sentinel configuration: %#v", configuration)
	}
}

func TestLoadAllowsDedicatedCacheCluster(t *testing.T) {
	configureSharedRedis(t)
	t.Setenv("REDIS_CACHE_MODE", "cluster")
	t.Setenv("REDIS_CACHE_ADDRESSES", "cache-1:6379,cache-2:6379,cache-3:6379")
	t.Setenv("REDIS_CACHE_DATABASE", "0")

	configuration, err := redisconnection.Load("cache")
	if err != nil {
		t.Fatal(err)
	}
	options := configuration.Options()
	if configuration.Mode != redisconnection.ModeCluster || !options.IsClusterMode || len(options.Addrs) != 3 {
		t.Fatalf("unexpected cluster configuration: %#v", configuration)
	}
}

func TestLoadRejectsNonzeroClusterDatabase(t *testing.T) {
	configureSharedRedis(t)
	t.Setenv("REDIS_QUEUE_MODE", "cluster")
	t.Setenv("REDIS_QUEUE_ADDRESSES", "queue-1:6379,queue-2:6379")
	t.Setenv("REDIS_QUEUE_DATABASE", "2")

	if _, err := redisconnection.Load("queue"); err == nil ||
		err.Error() != "REDIS_QUEUE_DATABASE must be 0 in cluster mode" {
		t.Fatalf("unexpected cluster database validation: %v", err)
	}
}

func TestLoadRejectsSentinelWithoutMasterName(t *testing.T) {
	configureSharedRedis(t)
	t.Setenv("REDIS_QUEUE_MODE", "sentinel")
	t.Setenv("REDIS_QUEUE_ADDRESSES", "sentinel-1:26379")
	t.Setenv("REDIS_QUEUE_MASTER_NAME", "")

	if _, err := redisconnection.Load("queue"); err == nil ||
		err.Error() != "REDIS_QUEUE_MASTER_NAME is required in sentinel mode" {
		t.Fatalf("unexpected Sentinel validation: %v", err)
	}
}

func TestConfiguredClientConnectsToRedis(t *testing.T) {
	if os.Getenv("REDIS_TOPOLOGY_INTEGRATION") != "1" {
		t.Skip("set REDIS_TOPOLOGY_INTEGRATION=1 to run the Redis topology integration test")
	}

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

func configureSharedRedis(t *testing.T) {
	t.Helper()
	t.Setenv("REDIS_MODE", "single")
	t.Setenv("REDIS_ADDRESSES", "")
	t.Setenv("REDIS_HOST", "shared-redis")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("REDIS_DATABASE", "0")
	t.Setenv("REDIS_USERNAME", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_MASTER_NAME", "")
	t.Setenv("REDIS_SENTINEL_USERNAME", "")
	t.Setenv("REDIS_SENTINEL_PASSWORD", "")
	t.Setenv("REDIS_TLS", "false")
}
