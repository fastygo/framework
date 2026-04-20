package cache

import (
	"hash/fnv"
	"sync"
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
	ttl    time.Duration
	shards [cacheShards]cacheShard[V]
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
// exit; suitable only for fixed-size lookup tables).
func New[V any](ttl time.Duration) *Cache[V] {
	c := &Cache[V]{ttl: ttl}
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
		delete(shard.items, key)
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
			}
		}
		shard.mu.Unlock()
	}
}
