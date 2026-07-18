package middleware

import (
	"net/http"
	"os"
	"strings"

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
// Set CORS_ALLOWED_ORIGIN to a comma-separated list of allowed origins
// (e.g. "http://localhost:5173,http://localhost:5174") or a single origin.
// When empty or "*", allows all origins (credentials disabled).
func CORSWrapper() gin.HandlerFunc {
	allowedOriginRaw := os.Getenv("CORS_ALLOWED_ORIGIN")
	allowedOrigins := parseOrigins(allowedOriginRaw)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		useCredentials := false
		if len(allowedOrigins) == 0 {
			// Open CORS — any origin, no credentials.
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			if matched := matchOrigin(allowedOrigins, origin); matched != "" {
				c.Header("Access-Control-Allow-Origin", matched)
				c.Header("Access-Control-Allow-Credentials", "true")
				useCredentials = true
			} else {
				// Origin not in allowlist: reject preflight and also non-OPTIONS requests
				// with an explicit 403 to ensure monitoring tools and load balancers can
				// accurately distinguish blocked cross-origin requests.
				if c.Request.Method == http.MethodOptions {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":       403,
					"msg":        "不允许的请求来源",
					"data":       nil,
					"request_id": requestID(c),
				})
				return
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Client-App")
		if useCredentials {
			c.Header("Access-Control-Expose-Headers", "Set-Cookie")
		}
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func parseOrigins(raw string) []string {
	if raw == "" || raw == "*" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var origins []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			origins = append(origins, p)
		}
	}
	return origins
}

func matchOrigin(allowed []string, origin string) string {
	for _, o := range allowed {
		if strings.EqualFold(o, origin) {
			return origin
		}
	}
	return ""
}
