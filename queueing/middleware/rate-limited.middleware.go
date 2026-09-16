// Package middleware holds reusable queue job middleware.
package middleware

import (
	"context"
	"errors"
	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/queueing/job"
	"github.com/gonstruct/core/ratelimit"
	"time"

	"github.com/rs/zerolog/log"
)

// RateLimited is a queue job middleware modelled on Laravel's
// Illuminate\Queue\Middleware\RateLimited. Each job run counts as one attempt
// against the keyed window. When the window is spent the job is released back
// onto the queue without counting a failed attempt, so the worker stays free
// instead of blocking. It also releases the job when the work reports a
// downstream rate limit (an error exposing RetryAfter), so a provider 429
// backs off cleanly instead of exhausting retries.
type RateLimited struct {
	key     string
	limit   ratelimit.Limit
	limiter *ratelimit.Limiter
}

// defaultLimiter is shared across every RateLimited instance: Middlewares() is
// re-evaluated on each job execution, so constructing a limiter (and with it a
// cache engine and redis client) per instance would leak a connection pool per
// run and, on the memory driver, reset the window every run.
var defaultLimiter = ratelimit.New(cache.New())

func NewRateLimited(key string, limit ratelimit.Limit) *RateLimited {
	return &RateLimited{key: key, limit: limit, limiter: defaultLimiter}
}

func (self *RateLimited) Handle(context context.Context, _ job.Job, next func(context.Context) error) error {
	tooMany, err := self.limiter.TooManyAttempts(context, self.key, self.limit)
	if err != nil {
		log.Warn().Ctx(context).Err(err).Str("limiter", self.key).Msg("[queueing] rate limiter check failed, executing fail-open")
	}

	if tooMany {
		wait, _ := self.limiter.AvailableIn(context, self.key)
		return job.SilentRelease(wait)
	}

	if _, err := self.limiter.Hit(context, self.key, self.limit); err != nil {
		log.Warn().Ctx(context).Err(err).Str("limiter", self.key).Msg("[queueing] rate limiter hit failed, executing fail-open")
	}

	runErr := next(context)
	if retryAfter, ok := retryAfterFrom(runErr); ok {
		return job.SilentRelease(retryAfter)
	}

	return runErr
}

// retryAfterFrom unwraps an error chain looking for a downstream rate-limit
// signal that carries its own backoff.
func retryAfterFrom(err error) (time.Duration, bool) {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if retryable, ok := current.(interface{ RetryAfter() time.Duration }); ok {
			return retryable.RetryAfter(), true
		}
	}

	return 0, false
}
