package cache

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestMemoryGet(t *testing.T) {
	start := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		key       string
		elapsed   time.Duration
		wantValue string
		wantFresh bool
		wantOK    bool
	}{
		{name: "fresh right after set", key: "selic", elapsed: 0, wantValue: "v1", wantFresh: true, wantOK: true},
		{name: "fresh just before ttl", key: "selic", elapsed: time.Hour - time.Nanosecond, wantValue: "v1", wantFresh: true, wantOK: true},
		{name: "stale at ttl but still returned", key: "selic", elapsed: time.Hour, wantValue: "v1", wantFresh: false, wantOK: true},
		{name: "stale long after ttl", key: "selic", elapsed: 72 * time.Hour, wantValue: "v1", wantFresh: false, wantOK: true},
		{name: "missing key", key: "ipca", elapsed: 0, wantValue: "", wantFresh: false, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := start
			c := NewMemory[string](time.Hour, func() time.Time { return now })
			c.Set("selic", "v1")
			now = now.Add(tt.elapsed)

			value, fresh, ok := c.Get(tt.key)
			if value != tt.wantValue || fresh != tt.wantFresh || ok != tt.wantOK {
				t.Errorf("Get(%q) = (%q, %v, %v), want (%q, %v, %v)",
					tt.key, value, fresh, ok, tt.wantValue, tt.wantFresh, tt.wantOK)
			}
		})
	}
}

func TestMemorySetRefreshesEntry(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := NewMemory[int](time.Minute, func() time.Time { return now })
	c.Set("k", 1)
	now = now.Add(2 * time.Minute)
	c.Set("k", 2)

	value, fresh, ok := c.Get("k")
	if value != 2 || !fresh || !ok {
		t.Errorf("Get after overwrite = (%d, %v, %v), want (2, true, true)", value, fresh, ok)
	}
}

// Run with -race to catch unsynchronized access.
func TestMemoryConcurrentAccess(t *testing.T) {
	c := NewMemory[int](time.Minute, time.Now)
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			key := strconv.Itoa(i % 5)
			c.Set(key, i)
			c.Get(key)
		})
	}
	wg.Wait()
}
