package cache_test

import (
	"errors"
	"testing"
	"time"

	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/config"
)

func TestACacheNeedsAConfig(t *testing.T) {
	if _, _, err := cache.New().Get(t.Context(), "key"); !errors.Is(err, cache.ErrNoConfig) {
		t.Fatalf("expected ErrNoConfig, got %v", err)
	}
}

func TestAnUnknownDriverIsRefused(t *testing.T) {
	if _, err := cache.NewEngine(config.Cache{Driver: "floppy"}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestTheMemoryDriverKeepsWhatItIsGiven(t *testing.T) {
	kept := cache.New(cache.WithConfig(config.Cache{Driver: config.MemoryCacheDriver}))
	if err := kept.Set(t.Context(), "key", "value", time.Minute); err != nil {
		t.Fatal(err)
	}

	if value, found, err := kept.Get(t.Context(), "key"); err != nil || !found || value != "value" {
		t.Fatalf("expected the value back, got %q %v (%v)", value, found, err)
	}
}
