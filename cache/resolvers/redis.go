package resolvers

import (
	"context"
	"errors"
	redisconnection "github.com/gonstruct/core/redis"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis is a cache resolver backed by a shared go-redis client. The client is
// initialised lazily on first use so the package can be imported safely from
// anywhere without requiring an explicit boot step.
type Redis struct {
	clientOnce sync.Once
	client     redis.UniversalClient
	clientErr  error
}

func NewRedis() *Redis {
	return &Redis{}
}

func (r *Redis) Client() (redis.UniversalClient, error) {
	r.clientOnce.Do(func() {
		r.client, r.clientErr = redisconnection.NewClient(redisconnection.DefaultConnection)
	})
	return r.client, r.clientErr
}

func (r *Redis) Get(ctx context.Context, key string) (string, bool, error) {
	client, err := r.Client()
	if err != nil {
		return "", false, err
	}

	value, err := client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	return value, true, nil
}

func (r *Redis) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	client, err := r.Client()
	if err != nil {
		return err
	}

	return client.Set(ctx, key, value, ttl).Err()
}

func (r *Redis) Delete(ctx context.Context, key string) error {
	client, err := r.Client()
	if err != nil {
		return err
	}

	return client.Del(ctx, key).Err()
}

func (r *Redis) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	client, err := r.Client()
	if err != nil {
		return 0, err
	}

	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	if count == 1 && ttl > 0 {
		if err := client.Expire(ctx, key, ttl).Err(); err != nil {
			return count, err
		}
	}

	return count, nil
}

func (r *Redis) Add(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	client, err := r.Client()
	if err != nil {
		return false, err
	}

	return client.SetNX(ctx, key, value, ttl).Result()
}
