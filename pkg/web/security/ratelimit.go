package security

import (
	"hash/fnv"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/fastygo/framework/pkg/web/middleware"
)

const rateLimiterShards = 64

type RateLimiter struct {
	rate      float64
	burst     float64
	shards    [rateLimiterShards]rateLimiterShard
}

type rateLimiterShard struct {
	mu       sync.Mutex
	visitors map[string]*rateLimiterVisitor
}

type rateLimiterVisitor struct {
	tokens   float64
	lastSeen time.Time
}

func NewRateLimiter(rate float64, burst int) *RateLimiter {
	if rate <= 0 {
		rate = 1
	}
	if burst <= 0 {
		burst = 1
	}

	rl := &RateLimiter{
		rate:  rate,
		burst: float64(burst),
	}
	for i := 0; i < rateLimiterShards; i++ {
		rl.shards[i] = rateLimiterShard{visitors: make(map[string]*rateLimiterVisitor)}
	}
	return rl
}

func (r *RateLimiter) shardFor(value string) *rateLimiterShard {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(value))
	index := int(hasher.Sum32() % rateLimiterShards)
	return &r.shards[index]
}

func (r *RateLimiter) Allow(ip string) bool {
	shard := r.shardFor(ip)
	now := time.Now()

	shard.mu.Lock()
	defer shard.mu.Unlock()

	visitor, exists := shard.visitors[ip]
	if !exists {
		visitor = &rateLimiterVisitor{
			tokens:   r.burst - 1,
			lastSeen: now,
		}
		shard.visitors[ip] = visitor
		return true
	}

	elapsed := now.Sub(visitor.lastSeen).Seconds()
	visitor.lastSeen = now
	visitor.tokens = math.Min(r.burst, visitor.tokens+elapsed*r.rate)
	if visitor.tokens >= 1 {
		visitor.tokens -= 1
		return true
	}
	return false
}

func (r *RateLimiter) Cleanup(staleAfter time.Duration) {
	now := time.Now()
	for i := range r.shards {
		shard := &r.shards[i]
		shard.mu.Lock()
		for key, visitor := range shard.visitors {
			if now.Sub(visitor.lastSeen) > staleAfter {
				delete(shard.visitors, key)
		}
		}
		shard.mu.Unlock()
	}
}

func RateLimitMiddleware(rl *RateLimiter, trustProxy bool) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rl == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := ClientIP(r, trustProxy)
			if ip == "" {
				ip = "unknown"
			}

			if !rl.Allow(ip) {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

