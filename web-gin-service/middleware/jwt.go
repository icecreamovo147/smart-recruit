package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	golangjwt "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     int32  `json:"role"`
	golangjwt.RegisteredClaims
}

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenText := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if tokenText == "" {
			c.AbortWithStatusJSON(http.StatusOK, gin.H{"code": 401, "msg": "未登录或 Token 无效", "data": nil, "request_id": requestID(c)})
			return
		}
		claims := &Claims{}
		token, err := golangjwt.ParseWithClaims(tokenText, claims, func(token *golangjwt.Token) (any, error) {
			// Explicitly require HS256 to prevent algorithm confusion attacks.
			if _, ok := token.Method.(*golangjwt.SigningMethodHMAC); !ok {
				return nil, golangjwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		}, golangjwt.WithValidMethods([]string{"HS256"}), golangjwt.WithLeeway(30*time.Second))
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusOK, gin.H{"code": 401, "msg": "未登录或 Token 无效", "data": nil, "request_id": requestID(c)})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func requestID(c *gin.Context) string {
	value, _ := c.Get("request_id")
	requestID, _ := value.(string)
	return requestID
}

func UserID(c *gin.Context) int64 {
	value, _ := c.Get("user_id")
	userID, _ := value.(int64)
	return userID
}

func Role(c *gin.Context) int32 {
	value, _ := c.Get("role")
	role, _ := value.(int32)
	return role
}
