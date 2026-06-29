package tft

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ipRateLimiter is a per-client-IP token-bucket limiter.
//
// Each IP gets its own bucket that refills at rps tokens/second up to burst
// tokens. A request consumes one token; when the bucket is empty the request
// is rejected. Idle buckets are swept periodically so memory stays bounded on
// a long-running server.
//
// It is implemented in-house (rather than pulling in a rate-limit library) to
// keep the dependency surface small; the logic is small enough to read and test.
type ipRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rps     float64
	burst   float64
	ttl     time.Duration
	now     func() time.Time // injectable for tests
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

func newIPRateLimiter(rps float64, burst int, ttl time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		buckets: make(map[string]*tokenBucket),
		rps:     rps,
		burst:   float64(burst),
		ttl:     ttl,
		now:     time.Now,
	}
}

// allow reports whether a request from ip may proceed, consuming a token if so.
func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	bucket, ok := l.buckets[ip]
	if !ok {
		// First request from this IP: start full, then spend one token.
		l.buckets[ip] = &tokenBucket{tokens: l.burst - 1, lastRefill: now, lastSeen: now}
		return true
	}

	// Refill based on elapsed time, capped at burst.
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * l.rps
	if bucket.tokens > l.burst {
		bucket.tokens = l.burst
	}
	bucket.lastRefill = now
	bucket.lastSeen = now

	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	return true
}

// sweep drops buckets that have been idle longer than the TTL.
func (l *ipRateLimiter) sweep() {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := l.now()
	for ip, bucket := range l.buckets {
		if cutoff.Sub(bucket.lastSeen) > l.ttl {
			delete(l.buckets, ip)
		}
	}
}

// RateLimitMiddleware returns a gin middleware that limits each client IP to
// rps requests/second with a burst allowance. A non-positive rps disables the
// limiter (the middleware becomes a passthrough), so callers can wire it
// unconditionally and control it by config.
func RateLimitMiddleware(rps float64, burst int) gin.HandlerFunc {
	if rps <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	if burst < 1 {
		burst = 1
	}

	limiter := newIPRateLimiter(rps, burst, 10*time.Minute)

	// Periodically evict idle IPs. The ticker lives for the process lifetime,
	// which matches the server's lifetime.
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.sweep()
		}
	}()

	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "rate limit exceeded, slow down",
			})
			return
		}
		c.Next()
	}
}
