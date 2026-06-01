package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"web-gin-service/pkg/logger"
)

const redisTokenBucketScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local capacity = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

local bucket = redis.call("HMGET", key, "tokens", "ts")
local tokens = tonumber(bucket[1])
local ts = tonumber(bucket[2])

if tokens == nil then
  tokens = capacity
  ts = now
end

local delta = math.max(0, now - ts) / 1000
tokens = math.min(capacity, tokens + (delta * rate))

local allowed = 0
if tokens >= 1 then
  allowed = 1
  tokens = tokens - 1
end

redis.call("HMSET", key, "tokens", tokens, "ts", now)
redis.call("PEXPIRE", key, ttl)
return allowed
`

// keyLimiter stores one rate.Limiter per rate-limit key (user-ID or IP).
type keyLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// InMemoryLimiter holds an in-memory per-key rate limiter with a controllable
// lifecycle. Call Close() to terminate its background cleanup goroutine.
type InMemoryLimiter struct {
	mu     sync.Mutex
	keys   map[string]*keyLimiter
	stopCh chan struct{}
	done   chan struct{}
	rps    rate.Limit
	burst  int
}

// NewInMemoryLimiter creates an in-memory per-key rate limiter.
// Call Close() to stop the background cleanup goroutine.
func NewInMemoryLimiter(rps rate.Limit, burst int) *InMemoryLimiter {
	l := &InMemoryLimiter{
		keys:   make(map[string]*keyLimiter),
		stopCh: make(chan struct{}),
		done:   make(chan struct{}),
		rps:    rps,
		burst:  burst,
	}
	go l.cleanupLoop()
	return l
}

func (l *InMemoryLimiter) cleanupLoop() {
	defer close(l.done)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-l.stopCh:
			return
		case <-ticker.C:
			l.mu.Lock()
			for key, entry := range l.keys {
				if time.Since(entry.lastSeen) > 10*time.Minute {
					delete(l.keys, key)
				}
			}
			l.mu.Unlock()
		}
	}
}

// Close signals the cleanup goroutine to exit and waits for it to finish.
func (l *InMemoryLimiter) Close() {
	close(l.stopCh)
	<-l.done
}

// Allow checks whether a request for the given key is permitted.
func (l *InMemoryLimiter) Allow(key string) bool {
	l.mu.Lock()
	entry, exists := l.keys[key]
	if !exists {
		entry = &keyLimiter{limiter: rate.NewLimiter(l.rps, l.burst)}
		l.keys[key] = entry
	}
	entry.lastSeen = time.Now()
	l.mu.Unlock()
	return entry.limiter.Allow()
}

// LimiterRegistry holds references to all in-memory rate limiters so the caller can
// call Close() during shutdown.
type LimiterRegistry struct {
	Auth    *InMemoryLimiter
	AI      *InMemoryLimiter
	General *InMemoryLimiter
}

// NewLimiterRegistry creates limiters for the three rate-limit scopes.
func NewLimiterRegistry(authRPS, authBurst, aiRPS, aiBurst, generalRPS, generalBurst int) *LimiterRegistry {
	return &LimiterRegistry{
		Auth:    NewInMemoryLimiter(rate.Limit(authRPS), authBurst),
		AI:      NewInMemoryLimiter(rate.Limit(aiRPS), aiBurst),
		General: NewInMemoryLimiter(rate.Limit(generalRPS), generalBurst),
	}
}

func (r *LimiterRegistry) Close() {
	if r.Auth != nil {
		r.Auth.Close()
	}
	if r.AI != nil {
		r.AI.Close()
	}
	if r.General != nil {
		r.General.Close()
	}
}

// ResilientRateLimit is a rate-limiting middleware that defaults to Redis distributed
// token-bucket limiting, but falls back to the provided in-memory InMemoryLimiter when Redis
// is unavailable, misbehaves, or returns an error. In the fallback path the limiter
// returns 429 when burst is exceeded (not 503), and logs a warning.
func ResilientRateLimit(rdb *redis.Client, scope string, rps, burst int, fallback *InMemoryLimiter) gin.HandlerFunc {
	script := redis.NewScript(redisTokenBucketScript)
	if rps <= 0 {
		rps = 1
	}
	if burst <= 0 {
		burst = rps
	}
	ttl := time.Duration(burst/rps+2) * time.Second
	if ttl < 3*time.Second {
		ttl = 3 * time.Second
	}

	return func(c *gin.Context) {
		key := rateLimitKey(c, scope)

		// Fast path: try Redis if the client is available.
		if rdb != nil {
			now := time.Now().UnixMilli()
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			result, err := script.Run(ctx, rdb, []string{key}, now, rps, burst, ttl.Milliseconds()).Int()
			cancel()
			if err == nil {
				if result == 1 {
					c.Next()
					return
				}
				// Burst exceeded on Redis path — return 429.
				rateLimitExceeded(c, scope, key, rps, burst)
				return
			}
			// Redis error — fall through to in-memory fallback.
			logger.L().Warn("redis rate limit unavailable, falling back to in-memory",
				zap.String("scope", scope),
				zap.String("key", key),
				zap.Error(err),
			)
		}

		// Fallback: in-memory limiter keyed by the same rateLimitKey.
		if !fallback.Allow(key) {
			rateLimitExceeded(c, scope, key, rps, burst)
			return
		}
		c.Next()
	}
}

// RedisRateLimit is kept for backward compatibility. New call sites should use
// ResilientRateLimit instead.
func RedisRateLimit(rdb *redis.Client, scope string, rps, burst int) gin.HandlerFunc {
	return ResilientRateLimit(rdb, scope, rps, burst, NewInMemoryLimiter(rate.Limit(rps), burst))
}

func rateLimitKey(c *gin.Context, scope string) string {
	if userID := UserID(c); userID > 0 {
		return "rate:" + scope + ":user:" + strconv.FormatInt(userID, 10)
	}
	return "rate:" + scope + ":ip:" + c.ClientIP()
}

func rateLimitExceeded(c *gin.Context, scope, key string, rps, burst int) {
	logger.L().Warn("rate limit exceeded",
		zap.String("request_id", requestID(c)),
		zap.String("scope", scope),
		zap.String("client_ip", c.ClientIP()),
		zap.Int64("user_id", UserID(c)),
		zap.String("key", key),
		zap.Int("rps", rps),
		zap.Int("burst", burst),
	)
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"code":       429,
		"msg":        "请求过于频繁，请稍后重试",
		"data":       nil,
		"request_id": requestID(c),
	})
}

// abortRateLimitUnavailable returns 503. Retained for backward compatibility with ai_quota.go
// and resume_quota.go, which represent quota/risk controls (not simple RPS limiting) and
// intentionally fail-closed when Redis is unavailable.
func abortRateLimitUnavailable(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
		"code":       503,
		"msg":        "系统繁忙，请稍后重试",
		"data":       nil,
		"request_id": requestID(c),
	})
}
