package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const resumeQuotaScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])

local current = redis.call("INCR", key)
if current == 1 then
    redis.call("EXPIRE", key, ttl)
end
if current > limit then
    return {0, current, limit}
end
return {1, current, limit}
`

// ResumePresignQuota enforces per-user hourly and daily limits on resume presign requests.
func ResumePresignQuota(rdb *redis.Client, hourlyLimit, dailyLimit int) gin.HandlerFunc {
	return resumeQuota(rdb, "resume_presign", 42911, "简历上传过于频繁，请稍后再试", hourlyLimit, dailyLimit)
}

// ResumeConfirmQuota enforces per-user hourly and daily limits on resume confirm requests.
func ResumeConfirmQuota(rdb *redis.Client, hourlyLimit, dailyLimit int) gin.HandlerFunc {
	return resumeQuota(rdb, "resume_confirm", 42912, "简历上传确认过于频繁，请稍后再试", hourlyLimit, dailyLimit)
}

func resumeQuota(rdb *redis.Client, scope string, code int, msg string, hourlyLimit, dailyLimit int) gin.HandlerFunc {
	script := redis.NewScript(resumeQuotaScript)

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
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		if hourlyLimit > 0 {
			hourKey := fmt.Sprintf("quota:%s:hour:%d:%s", scope, userID, now.Format("2006010215"))
			hourTTL := secondsUntilNextHour(now)
			if !checkQuota(ctx, script, rdb, hourKey, hourlyLimit, hourTTL) {
				resetAt := nextHour(now).Format(time.RFC3339)
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"code": code,
					"msg":  msg,
					"data": gin.H{
						"quota_type": "hourly",
						"reset_at":   resetAt,
					},
					"request_id": requestID(c),
				})
				return
			}
		}

		if dailyLimit > 0 {
			dayKey := fmt.Sprintf("quota:%s:day:%d:%s", scope, userID, now.Format("20060102"))
			dayTTL := secondsUntilMidnight(now)
			if !checkQuota(ctx, script, rdb, dayKey, dailyLimit, dayTTL) {
				resetAt := midnight(now).Format(time.RFC3339)
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"code": code,
					"msg":  msg,
					"data": gin.H{
						"quota_type": "daily",
						"reset_at":   resetAt,
					},
					"request_id": requestID(c),
				})
				return
			}
		}

		// Anomaly detection: track presign-confirm ratio. Too many presigns without confirms is suspicious.
		if scope == "resume_presign" {
			presignKey := fmt.Sprintf("quota:resume_presign:count:%d:%s", userID, now.Format("2006010215"))
			rdb.Incr(ctx, presignKey)
			rdb.Expire(ctx, presignKey, 2*time.Hour)
			confirmKey := fmt.Sprintf("quota:resume_confirm:count:%d:%s", userID, now.Format("2006010215"))
			confirmCount, _ := rdb.Get(ctx, confirmKey).Int()
			presignCount, _ := rdb.Get(ctx, presignKey).Int()
			if presignCount > 5 && confirmCount == 0 {
				BlockUser(rdb, userID, 15*time.Minute)
			}
		}
		if scope == "resume_confirm" {
			confirmKey := fmt.Sprintf("quota:resume_confirm:count:%d:%s", userID, now.Format("2006010215"))
			rdb.Incr(ctx, confirmKey)
			rdb.Expire(ctx, confirmKey, 2*time.Hour)
		}

		c.Next()
	}
}

func checkQuota(ctx context.Context, script *redis.Script, rdb *redis.Client, key string, limit, ttl int) bool {
	result, err := script.Run(ctx, rdb, []string{key}, limit, ttl).Slice()
	if err != nil {
		return false
	}
	return int(result[0].(int64)) == 1
}

func secondsUntilNextHour(now time.Time) int {
	next := nextHour(now)
	return int(next.Sub(now).Seconds()) + 1
}

func nextHour(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, now.Hour()+1, 0, 0, 0, now.Location())
}
