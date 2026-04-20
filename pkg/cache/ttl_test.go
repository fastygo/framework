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

	var remaining int
	for i := range c.shards {
		c.shards[i].mu.RLock()
		remaining += len(c.shards[i].items)
		c.shards[i].mu.RUnlock()
	}
	if remaining != 0 {
		t.Fatalf("expected 0 entries after Cleanup, got %d", remaining)
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
}
