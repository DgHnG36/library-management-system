package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/middleware"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// TestRateLimitMiddleware_AdminBypass tests that admin users bypass rate limiting
func TestRateLimitMiddleware_AdminBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Note: This test uses a real Redis client. For production testing,
	// consider using miniredis or Redis test containers.
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Skip test if Redis is not available
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		t.Skip("Redis not available, skipping test")
	}

	rlm := middleware.NewRateLimitMiddleware(redisClient, 5, 60, logger.DefaultNewLogger())

	router := gin.New()
	hitHandler := false
	router.Use(func(c *gin.Context) {
		// Set the role before the rate limit middleware
		c.Set("role", "admin")
		rlm.Handle()(c)
	})
	router.GET("/api/test", func(c *gin.Context) {
		hitHandler = true
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")

	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	// Handler should be called, context should pass through
	assert.True(t, hitHandler, "expected handler to be called for admin user")
}

// TestRateLimitMiddleware_Headers tests that rate limit headers are set correctly
func TestRateLimitMiddleware_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		t.Skip("Redis not available, skipping test")
	}

	rlm := middleware.NewRateLimitMiddleware(redisClient, 10, 60, logger.DefaultNewLogger())

	router := gin.New()
	router.Use(rlm.Handle())
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, "10", res.Header().Get("X-Rate-Limit"), "expected X-Rate-Limit header")
	assert.NotEmpty(t, res.Header().Get("X-Rate-Limit-Remaining"), "expected X-Rate-Limit-Remaining header")
}

// TestRateLimitMiddleware_LimitExceeded tests that 429 is returned when limit is exceeded
func TestRateLimitMiddleware_LimitExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		t.Skip("Redis not available, skipping test")
	}

	rlm := middleware.NewRateLimitMiddleware(redisClient, 2, 60, logger.DefaultNewLogger())

	router := gin.New()
	router.Use(rlm.Handle())
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Clear any existing rate limit keys for httptest default IP (192.0.2.1)
	ctx := context.Background()
	redisClient.Del(ctx, "ratelimit:192.0.2.1")

	// Make 3 requests (limit is 2)
	// httptest defaults to 192.0.2.1 as the remote address
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)

		if i < 2 {
			assert.Equal(t, http.StatusOK, res.Code, "request %d should succeed", i+1)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, res.Code, "request %d should be rate limited", i+1)
		}
	}
}
