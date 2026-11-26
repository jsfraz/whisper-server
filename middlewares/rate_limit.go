package middlewares

import (
	"jsfraz/shept-server/utils"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateBucket is a token bucket for one key (client IP).
type rateBucket struct {
	tokens   float64
	lastFill time.Time
}

// rateLimiter is a per-key token bucket kept in memory. One process, one VPS: a
// shared store would be more code than the threat needs.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	// burst is the bucket size, refill the number of tokens added per second.
	burst  float64
	refill float64
	// lastSweep bounds the map size by dropping idle buckets.
	lastSweep time.Time
}

func newRateLimiter(burst int, refillPerMinute int) *rateLimiter {
	return &rateLimiter{
		buckets:   make(map[string]*rateBucket),
		burst:     float64(burst),
		refill:    float64(refillPerMinute) / 60.0,
		lastSweep: time.Now(),
	}
}

// allow consumes one token for key and reports whether the call may proceed.
func (l *rateLimiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastSweep) > 5*time.Minute {
		for k, b := range l.buckets {
			if now.Sub(b.lastFill) > 5*time.Minute {
				delete(l.buckets, k)
			}
		}
		l.lastSweep = now
	}

	b, ok := l.buckets[key]
	if !ok {
		b = &rateBucket{tokens: l.burst, lastFill: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.lastFill).Seconds() * l.refill
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.lastFill = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// RateLimit rejects a client that exceeds burst requests and then refills at
// refillPerMinute. Used on the endpoints that cost CPU (login) or disk (upload).
//
//	@param burst
//	@param refillPerMinute
//	@return gin.HandlerFunc
func RateLimit(burst int, refillPerMinute int) gin.HandlerFunc {
	limiter := newRateLimiter(burst, refillPerMinute)
	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP()) {
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": utils.ErrorTooManyRequests})
			c.Abort()
			return
		}
		c.Next()
	}
}
