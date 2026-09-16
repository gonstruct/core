package resolvers

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// Memory is an in-process cache resolver intended for the memory driver and
// tests. Values expire lazily on read.
type Memory struct {
	mu      sync.Mutex
	entries map[string]memoryEntry
}

type memoryEntry struct {
	value     string
	expiresAt time.Time
}

// NewMemory returns an in-process cache. Entries expire lazily on read; the
// underlying map is not proactively swept, so workloads that write a very
// large number of distinct keys without ever reading them again will retain
// expired entries until overwritten. Intended for development and tests —
// production deployments should use the redis resolver.
func NewMemory() *Memory {
	return &Memory{entries: make(map[string]memoryEntry)}
}

func (m *Memory) Get(_ context.Context, key string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[key]
	if !ok {
		return "", false, nil
	}

	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		delete(m.entries, key)
		return "", false, nil
	}

	return entry.value, true, nil
}

func (m *Memory) Set(_ context.Context, key, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := memoryEntry{value: value}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	m.entries[key] = entry
	return nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries, key)
	return nil
}

func (m *Memory) Increment(_ context.Context, key string, ttl time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, live := m.liveEntry(key)

	count := int64(0)
	if live {
		count, _ = strconv.ParseInt(entry.value, 10, 64)
	}
	count++

	next := memoryEntry{value: strconv.FormatInt(count, 10)}
	switch {
	case live:
		next.expiresAt = entry.expiresAt
	case ttl > 0:
		next.expiresAt = time.Now().Add(ttl)
	}
	m.entries[key] = next

	return count, nil
}

func (m *Memory) Add(_ context.Context, key, value string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, live := m.liveEntry(key); live {
		return false, nil
	}

	entry := memoryEntry{value: value}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	m.entries[key] = entry

	return true, nil
}

func (m *Memory) liveEntry(key string) (memoryEntry, bool) {
	entry, ok := m.entries[key]
	if !ok {
		return memoryEntry{}, false
	}

	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		delete(m.entries, key)
		return memoryEntry{}, false
	}

	return entry, true
}
