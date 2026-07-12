package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes limits the request body size for JSON endpoints.
// 0 means no limit (use with caution — only for specific routes).
// Returns 413 if the body exceeds maxBytes, rather than silently truncating.
func MaxBodyBytes(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 {
			c.Next()
			return
		}
		// Wrap with http.MaxBytesReader to get a proper error (not silent truncation)
		// when the body exceeds the limit.
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes+1)
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"code":       413,
				"msg":        "请求体过大",
				"data":       nil,
				"request_id": requestID(c),
			})
			return
		}
		// MaxBytesReader returns an error if the body exceeds maxBytes.
		// We also double-check the length in case the underlying implementation
		// didn't produce an error for the exact-boundary case.
		if int64(len(body)) > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"code":       413,
				"msg":        "请求体过大",
				"data":       nil,
				"request_id": requestID(c),
			})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		c.Next()
	}
}
