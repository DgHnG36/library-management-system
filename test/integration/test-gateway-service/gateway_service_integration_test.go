package integration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/middleware"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/proxy"
	gatewayRouter "github.com/DgHnG36/lib-management-system/services/gateway-service/internal/router"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/alicebob/miniredis/v2"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

/* GATEWAY SERVICE TEST */
func TestGatewayService_MiddlewareOrder_CORSBeforeAuth(t *testing.T) {
	router := setupGatewayRouterForTarget(t, 5, "http://127.0.0.1:1")

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/books", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code)
}

func TestGatewayService_MiddlewareOrder_AuthBeforeRateLimit(t *testing.T) {
	router := setupGatewayRouterForTarget(t, 5, "http://127.0.0.1:1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code)
	assert.NotEmpty(t, res.Header().Get("X-Request-ID"))
	assert.NotEmpty(t, res.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, res.Header().Get("X-Rate-Limit"))
	assert.Empty(t, res.Header().Get("X-Rate-Limit-Remaining"))
}

func TestGatewayService_MiddlewareOrder_RateLimitAfterAuth(t *testing.T) {
	router := setupGatewayRouterForTarget(t, 1, "http://127.0.0.1:1")

	unReq := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	for i := 0; i < 5; i++ {
		res := httptest.NewRecorder()
		router.ServeHTTP(res, unReq)
		assert.Equal(t, http.StatusUnauthorized, res.Code)
	}

	token := mustGenerateValidToken(t)

	firstReq := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	firstReq.Header.Set("Authorization", "Bearer "+token)
	firstRes := newCloseNotifyRecorder()
	router.ServeHTTP(firstRes, firstReq)
	assert.NotEqual(t, http.StatusTooManyRequests, firstRes.Code)

	secondReq := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	secondReq.Header.Set("Authorization", "Bearer "+token)
	secondRes := httptest.NewRecorder()
	router.ServeHTTP(secondRes, secondReq)
	assert.Equal(t, http.StatusTooManyRequests, secondRes.Code)
}

func TestGatewayService_ReverseProxy_RouteAndForward(t *testing.T) {
	var forwardedPath string
	var forwardedQuery string
	var forwardedAuth string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedPath = r.URL.Path
		forwardedQuery = r.URL.RawQuery
		forwardedAuth = r.Header.Get("Authorization")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer backend.Close()

	router := setupGatewayRouterForTarget(t, 50, backend.URL)
	token := mustGenerateValidToken(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books?limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	res := newCloseNotifyRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "{\"ok\":true}", res.Body.String())
	assert.Equal(t, "/books", forwardedPath)
	assert.Equal(t, "limit=10", forwardedQuery)
	assert.Equal(t, "Bearer "+token, forwardedAuth)
}

func TestGatewayService_ReverseProxy_ForwardMethod(t *testing.T) {
	var forwardedMethod string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	router := setupGatewayRouterForTarget(t, 50, backend.URL)
	token := mustGenerateValidToken(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/books/42", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	res := newCloseNotifyRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, http.MethodDelete, forwardedMethod)
}

func TestGatewayService_ReverseProxy_ForwardPOSTBody(t *testing.T) {
	var forwardedMethod string
	var forwardedBody string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedMethod = r.Method
		bodyBytes, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		forwardedBody = string(bodyBytes)
		w.WriteHeader(http.StatusCreated)
	}))
	defer backend.Close()

	router := setupGatewayRouterForTarget(t, 50, backend.URL)
	token := mustGenerateValidToken(t)

	payload := `{"title":"Clean Architecture","author":"Robert C. Martin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/books", strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	res := newCloseNotifyRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusCreated, res.Code)
	assert.Equal(t, http.MethodPost, forwardedMethod)
	assert.Equal(t, payload, forwardedBody)
}

func TestGatewayService_ReverseProxy_ForwardMultipleRoutes(t *testing.T) {
	forwardedPaths := make([]string, 0)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedPaths = append(forwardedPaths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	router := setupGatewayRouterForTarget(t, 50, backend.URL)
	token := mustGenerateValidToken(t)

	routes := []struct {
		path         string
		expectedPath string
	}{
		{path: "/api/v1/books", expectedPath: "/books"},
		{path: "/api/v1/books/123", expectedPath: "/books/123"},
		{path: "/api/v1/users/42/orders", expectedPath: "/users/42/orders"},
		{path: "/api/v1/auth/login", expectedPath: "/auth/login"},
	}

	for _, route := range routes {
		req := httptest.NewRequest(http.MethodGet, route.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		res := newCloseNotifyRecorder()
		router.ServeHTTP(res, req)
		assert.Equal(t, http.StatusOK, res.Code)
	}

	assert.Len(t, forwardedPaths, len(routes))
	for i, route := range routes {
		assert.Equal(t, route.expectedPath, forwardedPaths[i])
	}
}

func TestGatewayService_ReverseProxy_ForwardAdditionalHeaders(t *testing.T) {
	var forwardedTraceID string
	var forwardedRequestID string
	var forwardedContentType string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedTraceID = r.Header.Get("X-Trace-ID")
		forwardedRequestID = r.Header.Get("X-Request-ID")
		forwardedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	router := setupGatewayRouterForTarget(t, 50, backend.URL)
	token := mustGenerateValidToken(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Trace-ID", "trace-123")
	req.Header.Set("X-Request-ID", "req-456")
	req.Header.Set("Content-Type", "application/json")

	res := newCloseNotifyRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "trace-123", forwardedTraceID)
	assert.Equal(t, "req-456", forwardedRequestID)
	assert.Equal(t, "application/json", forwardedContentType)
}

func TestGatewayService_ReverseProxy_TrailingSlash(t *testing.T) {
	var forwardedPath string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	router := setupGatewayRouterForTarget(t, 50, backend.URL)
	token := mustGenerateValidToken(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	res := newCloseNotifyRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "/books/", forwardedPath)
}

func TestRateLimit_ConcurrentRequests(t *testing.T) {
	router := setupGatewayRouterForTarget(t, 5, "http://127.0.0.1:1")
	token := mustGenerateValidToken(t)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
		}()
	}

	wg.Wait()
}

/* HELPER METHODS */

type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
	closeCh chan bool
}

func newCloseNotifyRecorder() *closeNotifyRecorder {
	return &closeNotifyRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		closeCh:          make(chan bool, 1),
	}
}

func (r *closeNotifyRecorder) CloseNotify() <-chan bool {
	return r.closeCh
}

func mustGenerateValidToken(t *testing.T) string {
	t.Helper()

	token, err := middleware.GenerateToken(
		"user-test",
		"user",
		"user@test.com",
		[]byte("integration-test-secret"),
		"HS256",
		15,
	)
	assert.NoError(t, err)

	return token
}

func setupGatewayRouterForTarget(t *testing.T, maxRequests int32, target string) http.Handler {
	t.Helper()

	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(redisServer.Close)

	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})

	log := logger.DefaultNewLogger()
	authMiddleware := middleware.NewAuthMiddleware([]byte("integration-test-secret"), "HS256", log)
	corsMiddleware := middleware.NewCORSMiddleware(
		[]string{"http://localhost:3000"},
		[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		[]string{"Origin", "Authorization", "Content-Type"},
		[]string{"X-Request-ID", "X-Rate-Limit", "X-Rate-Limit-Remaining"},
		true,
		12*time.Hour,
	)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(redisClient, maxRequests, 60, log)

	reverseProxy := proxy.NewReverseProxy(map[string]string{
		"/api/v1/auth": target,
		"/api/v1":      target,
	}, log)

	return gatewayRouter.SetupRouter(authMiddleware, corsMiddleware, rateLimitMiddleware, reverseProxy, log)
}
