package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RiskBlock checks Redis for temporary ban keys (risk:block:user:{id} and risk:block:ip:{ip}).
// Returns 42921 if the user or IP is temporarily blocked.
func RiskBlock(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		userID := UserID(c)
		if userID > 0 {
			userKey := fmt.Sprintf("risk:block:user:%d", userID)
			if exists, _ := rdb.Exists(ctx, userKey).Result(); exists > 0 {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"code":       42921,
					"msg":        "当前操作过于频繁，请稍后再试",
					"data":       nil,
					"request_id": requestID(c),
				})
				return
			}
		}

		ip := c.ClientIP()
		if ip != "" {
			ipKey := fmt.Sprintf("risk:block:ip:%s", ip)
			if exists, _ := rdb.Exists(ctx, ipKey).Result(); exists > 0 {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"code":       42921,
					"msg":        "当前操作过于频繁，请稍后再试",
					"data":       nil,
					"request_id": requestID(c),
				})
				return
			}
		}

		c.Next()
	}
}

// BlockUser sets a temporary ban on a user in Redis.
func BlockUser(rdb *redis.Client, userID int64, duration time.Duration) error {
	if rdb == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return rdb.Set(ctx, fmt.Sprintf("risk:block:user:%d", userID), "1", duration).Err()
}

// BlockIP sets a temporary ban on an IP in Redis.
func BlockIP(rdb *redis.Client, ip string, duration time.Duration) error {
	if rdb == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return rdb.Set(ctx, fmt.Sprintf("risk:block:ip:%s", ip), "1", duration).Err()
}
