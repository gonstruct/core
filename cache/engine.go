package cache

import (
	"context"
	"github.com/gonstruct/core/cache/resolvers"
	"time"

	"github.com/gonstruct/validation/env"
)

func NewEngine() (*Engine, error) {
	return &Engine{
		EngineResolver: map[string]EngineResolver{
			"memory": resolvers.NewMemory(),
			"redis":  resolvers.NewRedis(),
		}[env.String("CACHE_DRIVER")],
	}, nil
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
	Add(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
}
