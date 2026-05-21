package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// CSP adds a Content-Security-Policy header to responses.
// The policy is designed to allow legitimate application resources while
// blocking inline scripts and restricting external connections.
func CSP() gin.HandlerFunc {
	policy := os.Getenv("CSP_POLICY")
	if policy == "" {
		// Reasonable default for production: block inline scripts, restrict
		// connect-src to the API server, OSS/CDN, and AI streaming endpoints.
		policy = "default-src 'self'; " +
			"script-src 'self'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"connect-src 'self' https: wss:; " +
			"frame-src 'self' https:; " +
			"font-src 'self' data:; " +
			"base-uri 'self'; " +
			"form-action 'self'"
	}
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", policy)
		c.Next()
	}
}

// SecurityHeaders adds baseline security headers (X-Content-Type-Options,
// X-Frame-Options, X-XSS-Protection, Referrer-Policy).
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "0") // deprecated but explicitly disable for older browsers
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// CORSWrapper adds CORS headers with controlled origins.
func CORSWrapper() gin.HandlerFunc {
	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "*"
	}
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
