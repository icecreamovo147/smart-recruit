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

func JWTAuth(secret, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only consume Authorization if it's a Bearer header.
		// Other schemes (Basic, Digest, proxy-injected headers) must not
		// block the httpOnly cookie fallback.
		var tokenText string
		if authHeader := c.GetHeader("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			tokenText = authHeader[7:]
		}
		if tokenText == "" {
			if cookie, err := c.Cookie(cookieName); err == nil {
				tokenText = cookie
			}
		}
		if tokenText == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未登录或 Token 无效", "data": nil, "request_id": requestID(c)})
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未登录或 Token 无效", "data": nil, "request_id": requestID(c)})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func JWTAuthByClient(secret, candidateCookie, hrCookie, fallbackCookie string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookieName := fallbackCookie
		switch c.GetHeader("X-Client-App") {
		case "hr":
			cookieName = hrCookie
		case "candidate", "user":
			cookieName = candidateCookie
		}
		// Read from the short-lived access token cookie.
		JWTAuth(secret, cookieName+"_access")(c)
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
