package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

func TestResilientRateLimit_NilRedisFallsBackToMemory(t *testing.T) {
	t.Cleanup(func() { gin.SetMode(gin.DebugMode) })
	gin.SetMode(gin.TestMode)

	fallback := NewInMemoryLimiter(rate.Limit(1), 1)
	t.Cleanup(fallback.Close)
	router := rateLimitTestRouter(ResilientRateLimit(nil, "test", 1, 1, fallback))

	first := performRateLimitRequest(router)
	if first.Code != http.StatusOK {
		t.Fatalf("first request should pass via fallback, got %d", first.Code)
	}
	second := performRateLimitRequest(router)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request should be limited by fallback, got %d", second.Code)
	}
}

func TestResilientRateLimit_RedisErrorFallsBackToMemory(t *testing.T) {
	t.Cleanup(func() { gin.SetMode(gin.DebugMode) })
	gin.SetMode(gin.TestMode)

	rdb := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	_ = rdb.Close()
	t.Cleanup(func() { _ = rdb.Close() })
	fallback := NewInMemoryLimiter(rate.Limit(1), 1)
	t.Cleanup(fallback.Close)
	router := rateLimitTestRouter(ResilientRateLimit(rdb, "test", 1, 1, fallback))

	first := performRateLimitRequest(router)
	if first.Code != http.StatusOK {
		t.Fatalf("first request should pass via fallback after Redis error, got %d", first.Code)
	}
	second := performRateLimitRequest(router)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request should be limited by fallback after Redis error, got %d", second.Code)
	}
}

func rateLimitTestRouter(middleware gin.HandlerFunc) *gin.Engine {
	router := gin.New()
	router.Use(middleware)
	router.GET("/limited", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return router
}

func performRateLimitRequest(router *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "198.51.100.10:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
