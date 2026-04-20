package cache_test

import (
	"fmt"
	"time"

	"github.com/fastygo/framework/pkg/cache"
)

// ExampleCache demonstrates the typical Get/Set lifecycle of a TTL
// cache. Hit, miss, and expiry are shown in the same sequence a
// handler would experience them.
func ExampleCache() {
	c := cache.New[string](100 * time.Millisecond)

	c.Set("user:42", "ada")

	if v, ok := c.Get("user:42"); ok {
		fmt.Println("hit:", v)
	}

	if _, ok := c.Get("user:99"); !ok {
		fmt.Println("miss: user:99")
	}

	// Wait past the TTL: the entry is removed lazily on the next Get.
	time.Sleep(150 * time.Millisecond)
	if _, ok := c.Get("user:42"); !ok {
		fmt.Println("expired: user:42")
	}

	// Output:
	// hit: ada
	// miss: user:99
	// expired: user:42
}

// ExampleCache_nilReceiver shows that a nil *Cache is a safe no-op:
// Set is ignored, Get always returns the zero value and false. This
// is the pattern features use to make caching opt-in without nil
// guards at every call site.
func ExampleCache_nilReceiver() {
	var c *cache.Cache[int]

	c.Set("anything", 1)

	v, ok := c.Get("anything")
	fmt.Println(v, ok)

	// Output:
	// 0 false
}

// ExampleCache_Cleanup shows the active eviction path used by
// app.CleanupTask: after the TTL elapses, Cleanup walks every shard
// and removes expired entries — bounding memory for write-heavy
// caches whose keys are never read again.
func ExampleCache_Cleanup() {
	c := cache.New[int](50 * time.Millisecond)
	c.Set("k1", 1)
	c.Set("k2", 2)

	time.Sleep(75 * time.Millisecond)
	c.Cleanup()

	_, ok1 := c.Get("k1")
	_, ok2 := c.Get("k2")
	fmt.Println("k1 present:", ok1)
	fmt.Println("k2 present:", ok2)

	// Output:
	// k1 present: false
	// k2 present: false
}
