package cache

import (
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

const cacheShards = 16

// Cache is a generic, sharded TTL map safe for concurrent use.
//
// Entries are distributed across cacheShards (16) buckets keyed by an
// FNV-1a hash of the string key, so unrelated keys rarely contend for
// the same lock. Reads use an RLock; writes use a Lock; expired
// entries are dropped on the read path (lazy eviction) and via the
// explicit Cleanup method (active eviction — call it from a worker
// task to keep memory bounded).
//
// The zero value is not usable. Construct via New.
type Cache[V any] struct {
	ttl        time.Duration
	maxEntries int
	entries    atomic.Int64
	shards     [cacheShards]cacheShard[V]
}

// Options configures optional cache limits. The zero value preserves the
// unbounded behavior of New.
type Options struct {
	// MaxEntries caps the number of distinct keys stored in the cache.
	// Updates to existing keys are still allowed when the cache is full.
	// Values <= 0 disable the cap.
	MaxEntries int
}

// Stats reports cache cardinality and configured limits. It intentionally
// does not estimate byte usage because value sizes are type-specific.
type Stats struct {
	Entries    int
	Shards     int
	TTL        time.Duration
	MaxEntries int
}

type cacheShard[V any] struct {
	mu    sync.RWMutex
	items map[string]cacheItem[V]
}

type cacheItem[V any] struct {
	value     V
	expiresAt time.Time
}

// New creates a sharded TTL cache.
//
// Important: New does NOT spawn a background cleanup goroutine. Expired
// entries are dropped lazily on Get, but keys that are written once and
// never read again will accumulate forever. To bound memory in
// long-running processes, schedule periodic Cleanup via the framework's
// worker service:
//
//	htmlCache := cache.New[[]byte](10 * time.Minute)
//	builder.AddBackgroundTask(app.CleanupTask("html-cache-cleanup", time.Minute, htmlCache))
//
// Pass ttl <= 0 to disable expiry entirely (entries live until process
// exit; suitable only for fixed-size lookup tables). Do not use raw
// user-controlled input as a key unless you also configure a cardinality
// budget and cleanup strategy.
func New[V any](ttl time.Duration) *Cache[V] {
	return NewWithOptions[V](ttl, Options{})
}

// NewWithOptions creates a sharded TTL cache with optional cardinality
// limits. When MaxEntries is reached, Set silently ignores new keys while
// still allowing updates to keys that are already present.
func NewWithOptions[V any](ttl time.Duration, opts Options) *Cache[V] {
	c := &Cache[V]{ttl: ttl}
	if opts.MaxEntries > 0 {
		c.maxEntries = opts.MaxEntries
	}
	for i := range c.shards {
		c.shards[i].items = make(map[string]cacheItem[V])
	}
	return c
}

func (c *Cache[V]) keyShard(key string) *cacheShard[V] {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(key))
	index := int(hasher.Sum32() % cacheShards)
	return &c.shards[index]
}

// Get returns the value for key and reports whether it was present.
// A nil receiver yields the zero value and false (so a Cache field
// can be left unset to disable caching without a nil-check at every
// call site). Expired entries are removed before being reported as
// absent.
func (c *Cache[V]) Get(key string) (V, bool) {
	var empty V
	if c == nil {
		return empty, false
	}

	shard := c.keyShard(key)
	now := time.Now()

	shard.mu.RLock()
	item, exists := shard.items[key]
	shard.mu.RUnlock()
	if !exists {
		return empty, false
	}
	if c.ttl > 0 && now.After(item.expiresAt) {
		shard.mu.Lock()
		if current, ok := shard.items[key]; ok && current.expiresAt.Equal(item.expiresAt) {
			delete(shard.items, key)
			c.releaseEntry()
		}
		shard.mu.Unlock()
		return empty, false
	}
	return item.value, true
}

// Set stores value under key with the cache's configured TTL. A nil
// receiver is a no-op; callers can therefore leave a Cache field
// unset to opt out without guarding every Set call.
func (c *Cache[V]) Set(key string, value V) {
	if c == nil {
		return
	}

	var expiresAt time.Time
	if c.ttl > 0 {
		expiresAt = time.Now().Add(c.ttl)
	}

	shard := c.keyShard(key)
	shard.mu.Lock()
	if _, exists := shard.items[key]; !exists && !c.reserveEntry() {
		shard.mu.Unlock()
		return
	}
	shard.items[key] = cacheItem[V]{value: value, expiresAt: expiresAt}
	shard.mu.Unlock()
}

// Cleanup walks every shard and drops expired entries. Cheap enough
// to call once a minute even for large caches; pair it with
// app.CleanupTask for a self-bounding cache. A nil receiver and a
// cache with TTL <= 0 (no expiry) are both no-ops.
func (c *Cache[V]) Cleanup() {
	if c == nil || c.ttl <= 0 {
		return
	}

	now := time.Now()
	for i := range c.shards {
		shard := &c.shards[i]
		shard.mu.Lock()
		for key, item := range shard.items {
			if now.After(item.expiresAt) {
				delete(shard.items, key)
				c.releaseEntry()
			}
		}
		shard.mu.Unlock()
	}
}

// Len returns the number of entries currently retained by the cache. Expired
// entries are counted until they are removed by Get or Cleanup.
func (c *Cache[V]) Len() int {
	if c == nil {
		return 0
	}
	return int(c.entries.Load())
}

// Stats returns cache cardinality and configured limits. A nil cache returns
// the zero Stats value.
func (c *Cache[V]) Stats() Stats {
	if c == nil {
		return Stats{}
	}
	return Stats{
		Entries:    c.Len(),
		Shards:     cacheShards,
		TTL:        c.ttl,
		MaxEntries: c.maxEntries,
	}
}

func (c *Cache[V]) reserveEntry() bool {
	if c.maxEntries <= 0 {
		c.entries.Add(1)
		return true
	}
	for {
		current := c.entries.Load()
		if current >= int64(c.maxEntries) {
			return false
		}
		if c.entries.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

func (c *Cache[V]) releaseEntry() {
	c.entries.Add(-1)
}
