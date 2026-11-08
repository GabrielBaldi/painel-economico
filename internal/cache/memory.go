// Package cache provides an in-memory cache with a time-to-live.
package cache

import (
	"sync"
	"time"
)

// Memory is a TTL cache safe for concurrent use. An entry older than the TTL
// is reported as not fresh but is kept, so callers can serve it when the
// source of truth is down. It never evicts, which is fine for a small, fixed
// set of keys such as one per indicator.
type Memory[V any] struct {
	ttl   time.Duration
	now   func() time.Time
	mu    sync.RWMutex
	items map[string]entry[V]
}

type entry[V any] struct {
	value    V
	storedAt time.Time
}

// NewMemory returns an empty cache whose entries stay fresh for ttl.
// now is the clock; pass time.Now outside of tests.
func NewMemory[V any](ttl time.Duration, now func() time.Time) *Memory[V] {
	return &Memory[V]{ttl: ttl, now: now, items: make(map[string]entry[V])}
}

// Get returns the value stored under key, whether it is younger than the TTL,
// and whether the key exists at all.
func (c *Memory[V]) Get(key string) (value V, fresh, ok bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return value, false, false
	}
	return e.value, c.now().Sub(e.storedAt) < c.ttl, true
}

// Set stores value under key, replacing any previous value.
func (c *Memory[V]) Set(key string, value V) {
	c.mu.Lock()
	c.items[key] = entry[V]{value: value, storedAt: c.now()}
	c.mu.Unlock()
}
