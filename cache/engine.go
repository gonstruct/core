package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/gonstruct/core/cache/resolvers"
	"github.com/gonstruct/core/config"
)

func NewEngine(cache config.Cache) (*Engine, error) {
	switch cache.Driver {
	case config.MemoryCacheDriver:
		return &Engine{EngineResolver: resolvers.NewMemory()}, nil
	case config.RedisCacheDriver:
		return &Engine{EngineResolver: resolvers.NewRedis(cache.Connection.String())}, nil
	default:
		return nil, fmt.Errorf("[cache] unknown cache driver %q", cache.Driver)
	}
}

type Engine struct {
	EngineResolver
}

// EngineResolver describes the minimal cache surface needed by the rest of
// the application. Resolvers can implement additional capabilities, but this
// interface is what the facade exposes.
type EngineResolver interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
	IncrementBy(ctx context.Context, key string, amount int64, ttl time.Duration) (int64, error)
	Add(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
}
