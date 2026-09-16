package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Cache struct {
	engine *Engine
	mu     sync.Mutex
}

func New(options ...Option) *Cache {
	cache := new(Cache)

	for _, option := range options {
		option(cache)
	}

	return cache
}

func (c *Cache) Get(ctx context.Context, key string) (string, bool, error) {
	if err := c.ensureEngine(); err != nil {
		return "", false, fmt.Errorf("[cache] cannot read key %q: %w", key, err)
	}
	return c.engine.Get(ctx, key)
}

func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := c.ensureEngine(); err != nil {
		return fmt.Errorf("[cache] cannot write key %q: %w", key, err)
	}
	return c.engine.Set(ctx, key, value, ttl)
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.ensureEngine(); err != nil {
		return fmt.Errorf("[cache] cannot delete key %q: %w", key, err)
	}
	return c.engine.Delete(ctx, key)
}

// Increment atomically bumps the counter at key, applying ttl on the first
// increment so the counter expires with its window.
func (c *Cache) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if err := c.ensureEngine(); err != nil {
		return 0, fmt.Errorf("[cache] cannot increment key %q: %w", key, err)
	}
	return c.engine.Increment(ctx, key, ttl)
}

// Add writes value at key only when it does not already exist, returning whether
// the write happened. Used for write-once markers such as a rate window timer.
func (c *Cache) Add(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	if err := c.ensureEngine(); err != nil {
		return false, fmt.Errorf("[cache] cannot add key %q: %w", key, err)
	}
	return c.engine.Add(ctx, key, value, ttl)
}

func (c *Cache) ensureEngine() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.engine != nil {
		return nil
	}

	engine, err := NewEngine()
	if err != nil {
		return err
	}

	if engine.EngineResolver == nil {
		return fmt.Errorf("no cache resolver configured (set CACHE_DRIVER)")
	}

	c.engine = engine
	return nil
}
