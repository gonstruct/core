// Package ratelimit is a cache-backed rate limiter modelled on Laravel's
// Illuminate\Cache\RateLimiter: attempts are counted per key in a shared cache
// store, so every worker counts against the same window and the limit holds
// across processes.
package ratelimit

import (
	"context"
	"strconv"
	"time"

	"github.com/gonstruct/core/cache"
)

// Limit is the maximum number of hits allowed within a decay window.
type Limit struct {
	MaxAttempts int
	Decay       time.Duration
}

// PerMinute builds a Limit of maxAttempts hits per minute.
func PerMinute(maxAttempts int) Limit {
	return Limit{MaxAttempts: maxAttempts, Decay: time.Minute}
}

type Limiter struct {
	cache *cache.Cache
}

func New(store *cache.Cache) *Limiter {
	return &Limiter{cache: store}
}

// TooManyAttempts reports whether the key has reached its limit for the current
// window.
func (self *Limiter) TooManyAttempts(context context.Context, key string, limit Limit) (bool, error) {
	attempts, err := self.attempts(context, key)
	if err != nil {
		return false, err
	}

	return attempts >= int64(limit.MaxAttempts), nil
}

// Hit records one attempt against the key and returns the running count for the
// current window. The window timer is written once per window so the reset time
// stays fixed rather than sliding on every hit.
//
// The counter is created with Add (an atomic set-if-absent carrying the TTL)
// before it is incremented, so the key is born with its expiry and a crash
// mid-sequence can never leave an immortal counter behind. When the key expires
// between the Add and the Increment, the increment recreates it bare — the
// count-of-one rewrite below re-attaches the window TTL.
func (self *Limiter) Hit(context context.Context, key string, limit Limit) (int64, error) {
	resetAt := strconv.FormatInt(time.Now().Add(limit.Decay).UnixMilli(), 10)
	if _, err := self.cache.Add(context, self.timerKey(key), resetAt, limit.Decay); err != nil {
		return 0, err
	}

	added, err := self.cache.Add(context, key, "0", limit.Decay)
	if err != nil {
		return 0, err
	}

	count, err := self.cache.Increment(context, key, limit.Decay)
	if err != nil {
		return 0, err
	}

	if !added && count == 1 {
		if err := self.cache.Set(context, key, "1", limit.Decay); err != nil {
			return count, err
		}
	}

	return count, nil
}

// AvailableIn returns how long until the key's window resets, or zero when no
// window is active.
func (self *Limiter) AvailableIn(context context.Context, key string) (time.Duration, error) {
	value, found, err := self.cache.Get(context, self.timerKey(key))
	if err != nil || !found {
		return 0, err
	}

	resetAt, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}

	remaining := time.Until(time.UnixMilli(resetAt))
	if remaining < 0 {
		return 0, nil
	}

	return remaining, nil
}

func (self *Limiter) attempts(context context.Context, key string) (int64, error) {
	value, found, err := self.cache.Get(context, key)
	if err != nil || !found {
		return 0, err
	}

	return strconv.ParseInt(value, 10, 64)
}

func (self *Limiter) timerKey(key string) string {
	return key + ":timer"
}
