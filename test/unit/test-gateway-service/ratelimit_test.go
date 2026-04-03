package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/middleware"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

/* HELPER */

func newTestRedisClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(func() { mr.Close() })

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return mr, client
}

func newRateLimitRouter(mw *middleware.RateLimitMiddleware, role string) *gin.Engine {
	router := gin.New()
	router.GET("/api/v1/orders", func(c *gin.Context) {
		if role != "" {
			c.Set("X-User-Role", role)
		}
	}, mw.Handle(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	return router
}

/* TESTCASE */

func TestRateLimit_AllowRequestUnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, client := newTestRedisClient(t)
	mw := middleware.NewRateLimitMiddleware(client, 5, 60, testLogger)
	router := newRateLimitRouter(mw, "")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code)
	assert.Equal(t, "5", res.Header().Get("X-Rate-Limit"))
	assert.Equal(t, "4", res.Header().Get("X-Rate-Limit-Remaining"))
}

func TestRateLimit_BlockRequestOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, client := newTestRedisClient(t)
	const maxRequests = 3
	mw := middleware.NewRateLimitMiddleware(client, maxRequests, 60, testLogger)
	router := newRateLimitRouter(mw, "")

	for i := 0; i < maxRequests; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		assert.Equal(t, http.StatusNoContent, res.Code, "request %d should pass", i+1)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusTooManyRequests, res.Code, "request over limit should return 429")
	assert.Equal(t, "0", res.Header().Get("X-Rate-Limit-Remaining"))
}

func TestRateLimit_AdminBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, client := newTestRedisClient(t)
	mw := middleware.NewRateLimitMiddleware(client, 1, 60, testLogger)
	router := newRateLimitRouter(mw, "ADMIN")

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		assert.Equal(t, http.StatusNoContent, res.Code, "admin request %d should bypass rate limit", i+1)
		assert.Empty(t, res.Header().Get("X-Rate-Limit"), "admin should not have rate limit headers")
	}
}

func TestRateLimit_RateLimitHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, client := newTestRedisClient(t)
	const maxRequests = 10
	mw := middleware.NewRateLimitMiddleware(client, maxRequests, 60, testLogger)
	router := newRateLimitRouter(mw, "")

	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)

		assert.Equal(t, "10", res.Header().Get("X-Rate-Limit"))
		assert.Equal(t, fmt.Sprintf("%d", maxRequests-i), res.Header().Get("X-Rate-Limit-Remaining"),
			"after %d requests, remaining should be %d", i, maxRequests-i)
	}
}

func TestRateLimit_RedisError_FailOpen(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr, client := newTestRedisClient(t)
	mr.Close() // force Redis unavailable

	mw := middleware.NewRateLimitMiddleware(client, 5, 60, testLogger)
	router := newRateLimitRouter(mw, "")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "should fail open when Redis is unavailable")
}

func TestRateLimit_DifferentRolesHaveSeparateCounters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, client := newTestRedisClient(t)
	const maxRequests = 2
	mw := middleware.NewRateLimitMiddleware(client, maxRequests, 60, testLogger)

	makeRouter := func(role string) *gin.Engine {
		return newRateLimitRouter(mw, role)
	}

	// Exhaust limit for "user" role
	userRouter := makeRouter("user")
	for i := 0; i < maxRequests; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
		res := httptest.NewRecorder()
		userRouter.ServeHTTP(res, req)
		assert.Equal(t, http.StatusNoContent, res.Code)
	}

	// "user" role is now over limit
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	res := httptest.NewRecorder()
	userRouter.ServeHTTP(res, req)
	assert.Equal(t, http.StatusTooManyRequests, res.Code, "user role should be rate limited")

	// "manager" role should still have its own separate counter
	managerRouter := makeRouter("manager")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	res = httptest.NewRecorder()
	managerRouter.ServeHTTP(res, req)
	assert.Equal(t, http.StatusNoContent, res.Code, "manager role should have its own counter")
}
