package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const aiDailyQuotaScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])

local current = tonumber(redis.call("GET", key) or "0")
if current >= limit then
    return {0, current, limit}
end
current = redis.call("INCR", key)
if current == 1 then
    redis.call("EXPIRE", key, ttl)
end
return {1, current, limit}
`

const aiQuotaConsumedKey = "smart_recruit:ai_quota_consumed"

// MarkAIQuotaConsumed records that an AI request passed entitlement/billing
// admission and entered the runtime. Pre-admission failures are refunded from
// the gateway request quota after the handler returns.
func MarkAIQuotaConsumed(c *gin.Context) {
	if c != nil {
		c.Set(aiQuotaConsumedKey, true)
	}
}

func MarkAIQuotaFailed(c *gin.Context) {
	if c != nil {
		if _, exists := c.Get(aiQuotaConsumedKey); !exists {
			c.Set(aiQuotaConsumedKey, false)
		}
	}
}

func aiQuotaRequestConsumed(c *gin.Context) bool {
	if c == nil {
		return false
	}
	if consumed, marked := c.Get(aiQuotaConsumedKey); marked {
		value, _ := consumed.(bool)
		return value
	}
	return c.Writer.Status() < http.StatusBadRequest
}

// AIDailyQuota enforces per-user daily AI call limits using a Redis counter.
// Scope should be "candidate" or "hr". The key auto-resets at midnight local time.
func AIDailyQuota(rdb *redis.Client, scope string, dailyLimit int) gin.HandlerFunc {
	script := redis.NewScript(aiDailyQuotaScript)
	if dailyLimit <= 0 {
		dailyLimit = 1
	}

	return func(c *gin.Context) {
		if rdb == nil {
			abortRateLimitUnavailable(c)
			return
		}
		userID := UserID(c)
		if userID <= 0 {
			c.Next()
			return
		}
		now := time.Now()
		dateKey := now.Format("20060102")
		key := fmt.Sprintf("quota:ai:daily:%s:%d:%s", scope, userID, dateKey)
		ttl := secondsUntilMidnight(now)

		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		result, err := script.Run(ctx, rdb, []string{key}, dailyLimit, ttl).Slice()
		cancel()
		if err != nil {
			abortRateLimitUnavailable(c)
			return
		}
		allowed := int(result[0].(int64))
		used := int(result[1].(int64))
		limit := int(result[2].(int64))
		if allowed != 1 {
			resetAt := midnight(now).Format(time.RFC3339)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": 42901,
				"msg":  "今日 AI 使用次数已达上限，请明天再试",
				"data": gin.H{
					"quota_type": "daily",
					"limit":      limit,
					"used":       used,
					"reset_at":   resetAt,
				},
				"request_id": requestID(c),
			})
			return
		}

		c.Next()
		accountingCtx, accountingCancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), time.Second)
		defer accountingCancel()
		if !aiQuotaRequestConsumed(c) {
			_ = rdb.Decr(accountingCtx, key).Err()
			return
		}

		// Anomaly detection counts only requests that passed AI admission. A denied
		// credit reservation must not create a ten-minute risk block.
		windowKey := fmt.Sprintf("quota:ai:window:%s:%d:%d", scope, userID, now.Unix()/600)
		windowTTL := 900
		windowCount, _ := rdb.Incr(accountingCtx, windowKey).Result()
		if windowCount == 1 {
			rdb.Expire(accountingCtx, windowKey, time.Duration(windowTTL)*time.Second)
		}
		windowThreshold := dailyLimit / 2
		if windowThreshold < 3 {
			windowThreshold = 3
		}
		if int(windowCount) > windowThreshold {
			BlockUser(rdb, userID, 10*time.Minute)
		}

	}
}

func secondsUntilMidnight(now time.Time) int {
	tomorrow := midnight(now).Add(24 * time.Hour)
	return int(tomorrow.Sub(now).Seconds()) + 1
}

func midnight(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}
