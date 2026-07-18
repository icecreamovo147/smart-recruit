package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"smart-recruit-gateway/pkg/contextkeys"
	"smart-recruit-gateway/pkg/logger"
	"smart-recruit-gateway/pkg/observability"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}
		trace := observability.NewTraceContext(c.GetHeader("traceparent"))
		c.Set("request_id", requestID)
		c.Set("trace_id", trace.TraceID)
		c.Set("span_id", trace.SpanID)
		c.Header("X-Request-ID", requestID)
		c.Header("traceparent", trace.Traceparent)
		c.Header("X-Trace-ID", trace.TraceID)
		c.Header("X-Span-ID", trace.SpanID)
		// Store in context.Context for gRPC metadata propagation.
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, contextkeys.RequestID, requestID)
		ctx = context.WithValue(ctx, contextkeys.ClientIP, c.ClientIP())
		ctx = context.WithValue(ctx, contextkeys.TraceID, trace.TraceID)
		ctx = context.WithValue(ctx, contextkeys.SpanID, trace.SpanID)
		ctx = context.WithValue(ctx, contextkeys.Traceparent, trace.Traceparent)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(bytes[:])
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestID, _ := c.Get("request_id")
				route := c.FullPath()
				if route == "" {
					route = "unmatched"
				}
				observability.DefaultMetrics.RecordHTTPPanic(c.Request.Method, route)
				logger.L().Error("panic recovered",
					zap.Any("request_id", requestID),
					zap.String("trace_id", traceID(c)),
					zap.Any("panic", recovered),
					zap.String("stack", string(debug.Stack())),
				)
				c.AbortWithStatusJSON(http.StatusOK, gin.H{"code": 500, "msg": "服务暂时不可用，请稍后重试", "data": nil, "request_id": requestID})
			}
		}()
		c.Next()
	}
}

func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		// Capture auth context if available (set by JWTAuth middleware).
		var userID int64
		if uid, ok := c.Get("user_id"); ok {
			userID, _ = uid.(int64)
		}
		logger.L().Info("http",
			zap.String("request_id", requestID(c)),
			zap.String("trace_id", traceID(c)),
			zap.String("span_id", spanID(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("cost", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int64("user_id", userID),
		)
	}
}

func Metrics(registry *observability.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		registry.ObserveHTTPRequest(c.Request.Method, route, c.Writer.Status(), time.Since(start))
	}
}

func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func traceID(c *gin.Context) string {
	if value, ok := c.Get("trace_id"); ok {
		if s, ok := value.(string); ok {
			return s
		}
	}
	return ""
}

func spanID(c *gin.Context) string {
	if value, ok := c.Get("span_id"); ok {
		if s, ok := value.(string); ok {
			return s
		}
	}
	return ""
}
