package ratelimit_test

import (
	"testing"
	"time"

	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/ratelimit"
)

func TestIncrementWeighsACallByItsAmount(t *testing.T) {
	t.Setenv("CACHE_DRIVER", "memory")
	limiter := ratelimit.New(cache.New())
	limit := ratelimit.PerMinute(10)
	ctx := t.Context()

	if count, err := limiter.Increment(ctx, "calls", limit, 4); err != nil || count != 4 {
		t.Fatalf("expected 4 after the first call, got %d (%v)", count, err)
	}
	if count, err := limiter.Hit(ctx, "calls", limit); err != nil || count != 5 {
		t.Fatalf("expected 5 after a hit, got %d (%v)", count, err)
	}
	if remaining, err := limiter.Remaining(ctx, "calls", limit); err != nil || remaining != 5 {
		t.Fatalf("expected 5 left, got %d (%v)", remaining, err)
	}
	if count, err := limiter.Increment(ctx, "calls", limit, 6); err != nil || count != 11 {
		t.Fatalf("expected 11, got %d (%v)", count, err)
	}
	if remaining, err := limiter.Remaining(ctx, "calls", limit); err != nil || remaining != 0 {
		t.Fatalf("an overspent window has none left, got %d (%v)", remaining, err)
	}
	if wait, err := limiter.AvailableIn(ctx, "calls"); err != nil || wait <= 0 || wait > time.Minute {
		t.Fatalf("the window should reset within the minute, got %s (%v)", wait, err)
	}
}
