package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit returns a per-IP rate limiting middleware using token bucket.
// rps = sustained requests per second, burst = max instantaneous burst.
func RateLimit(rps, burst int) gin.HandlerFunc {
	var mu sync.Mutex
	ips := make(map[string]*ipLimiter)

	// Periodic cleanup of idle IPs
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			for ip, l := range ips {
				if time.Since(l.lastSeen) > 10*time.Minute {
					delete(ips, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		entry, exists := ips[ip]
		if !exists {
			entry = &ipLimiter{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
			ips[ip] = entry
		}
		entry.lastSeen = time.Now()
		mu.Unlock()

		if !entry.limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":       429,
				"msg":        "请求过于频繁，请稍后重试",
				"data":       nil,
				"request_id": requestID(c),
			})
			return
		}
		c.Next()
	}
}

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

// RedisRateLimit applies a distributed token-bucket limiter. Redis errors are
// fail-closed by design: when Redis cannot make the decision, the request is
// rejected to protect downstream services.
func RedisRateLimit(rdb *redis.Client, scope string, rps, burst int) gin.HandlerFunc {
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
		if rdb == nil {
			abortRateLimitUnavailable(c)
			return
		}
		key := rateLimitKey(c, scope)
		now := time.Now().UnixMilli()
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		result, err := script.Run(ctx, rdb, []string{key}, now, rps, burst, ttl.Milliseconds()).Int()
		if err != nil {
			abortRateLimitUnavailable(c)
			return
		}
		if result != 1 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":       429,
				"msg":        "请求过于频繁，请稍后重试",
				"data":       nil,
				"request_id": requestID(c),
			})
			return
		}
		c.Next()
	}
}

func rateLimitKey(c *gin.Context, scope string) string {
	if userID := UserID(c); userID > 0 {
		return "rate:" + scope + ":user:" + stringID(userID)
	}
	return "rate:" + scope + ":ip:" + c.ClientIP()
}

func stringID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func abortRateLimitUnavailable(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
		"code":       503,
		"msg":        "系统繁忙，请稍后重试",
		"data":       nil,
		"request_id": requestID(c),
	})
}
