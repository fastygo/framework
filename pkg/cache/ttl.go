package cache

import (
	"hash/fnv"
	"sync"
	"time"
)

const cacheShards = 16

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
