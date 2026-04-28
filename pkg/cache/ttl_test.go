package cache

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	t.Parallel()

	c := New[string](time.Minute)
	c.Set("welcome", "value")
	if got := c.Len(); got != 1 {
		t.Fatalf("Len after Set = %d, want 1", got)
	}

	got, ok := c.Get("welcome")
	if !ok {
		t.Fatal("expected cached value to exist")
	}
	if got != "value" {
		t.Fatalf("expected value=%q, got %q", "value", got)
	}

	_, ok = c.Get("missing")
	if ok {
		t.Fatal("expected missing key to be absent")
	}
	if got := c.Len(); got != 1 {
		t.Fatalf("Len after miss = %d, want 1", got)
	}
}

func TestCacheTTLExpiry(t *testing.T) {
	t.Parallel()

	c := New[string](20 * time.Millisecond)
	c.Set("welcome", "value")
	time.Sleep(30 * time.Millisecond)

	_, ok := c.Get("welcome")
	if ok {
		t.Fatal("expected expired cache entry to be gone")
	}
	if got := c.Len(); got != 0 {
		t.Fatalf("Len after lazy expiry = %d, want 0", got)
	}
}

// TestCacheCleanupBoundsGrowth verifies that Cleanup actually evicts every
// expired entry across all shards, so a periodic call keeps memory bounded.
func TestCacheCleanupBoundsGrowth(t *testing.T) {
	t.Parallel()

	const total = 10_000
	c := New[int](20 * time.Millisecond)
	for i := 0; i < total; i++ {
		c.Set(strconv.Itoa(i), i)
	}

	time.Sleep(50 * time.Millisecond)
	c.Cleanup()

	if remaining := c.Len(); remaining != 0 {
		t.Fatalf("expected 0 entries after Cleanup, got %d", remaining)
	}
}

func TestCacheStats(t *testing.T) {
	t.Parallel()

	c := NewWithOptions[string](time.Minute, Options{MaxEntries: 2})
	c.Set("a", "one")
	c.Set("b", "two")

	stats := c.Stats()
	if stats.Entries != 2 {
		t.Fatalf("Stats Entries = %d, want 2", stats.Entries)
	}
	if stats.Shards != cacheShards {
		t.Fatalf("Stats Shards = %d, want %d", stats.Shards, cacheShards)
	}
	if stats.TTL != time.Minute {
		t.Fatalf("Stats TTL = %v, want %v", stats.TTL, time.Minute)
	}
	if stats.MaxEntries != 2 {
		t.Fatalf("Stats MaxEntries = %d, want 2", stats.MaxEntries)
	}

	var nilCache *Cache[string]
	if got := nilCache.Len(); got != 0 {
		t.Fatalf("nil Len = %d, want 0", got)
	}
	if got := nilCache.Stats(); got != (Stats{}) {
		t.Fatalf("nil Stats = %+v, want zero", got)
	}
}

func TestCacheMaxEntries(t *testing.T) {
	t.Parallel()

	c := NewWithOptions[string](time.Minute, Options{MaxEntries: 2})
	c.Set("a", "one")
	c.Set("b", "two")
	c.Set("c", "three")

	if got := c.Len(); got != 2 {
		t.Fatalf("Len with MaxEntries = %d, want 2", got)
	}
	if _, ok := c.Get("c"); ok {
		t.Fatal("new key beyond MaxEntries must be ignored")
	}

	c.Set("a", "updated")
	got, ok := c.Get("a")
	if !ok || got != "updated" {
		t.Fatalf("existing key update should be allowed when full, got %q ok=%v", got, ok)
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	t.Parallel()

	const total = 100
	c := New[int](time.Minute)
	var wg sync.WaitGroup

	wg.Add(total * 2)
	for i := 0; i < total; i++ {
		key := strconv.Itoa(i)
		go func(value int, k string) {
			defer wg.Done()
			c.Set(k, value)
		}(i, key)
		go func(k string) {
			defer wg.Done()
			_, _ = c.Get(k)
		}(key)
	}
	wg.Wait()

	for i := 0; i < total; i++ {
		key := fmt.Sprintf("%d", i)
		if _, ok := c.Get(key); !ok {
			t.Fatalf("expected key %q to exist", key)
		}
	}
	if got := c.Len(); got != total {
		t.Fatalf("Len after concurrent access = %d, want %d", got, total)
	}
}
