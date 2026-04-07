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
