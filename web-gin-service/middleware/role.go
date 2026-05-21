package middleware

import "github.com/gin-gonic/gin"

func RequireRole(role int32) gin.HandlerFunc {
	return func(c *gin.Context) {
		if Role(c) != role {
			c.AbortWithStatusJSON(200, gin.H{"code": 403, "msg": "无权限操作", "data": nil, "request_id": requestID(c)})
			return
		}
		c.Next()
	}
}

// RequireMinRole allows users with role >= minRole (e.g. both HR=2 and Admin=3).
func RequireMinRole(minRole int32) gin.HandlerFunc {
	return func(c *gin.Context) {
		if Role(c) < minRole {
			c.AbortWithStatusJSON(200, gin.H{"code": 403, "msg": "无权限操作", "data": nil, "request_id": requestID(c)})
			return
		}
		c.Next()
	}
}
