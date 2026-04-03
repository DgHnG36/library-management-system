package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewCORSMiddleware(12 * time.Hour)

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "expected status %d, got %d", http.StatusNoContent, res.Code)
	if !hitHandler {
		t.Fatalf("expected handler to be hit for allowed origin, but it was not")
	}

	assert.Equal(t, "http://localhost:5173", res.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, PATCH", res.Header().Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, res.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "true", res.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewCORSMiddleware(12 * time.Hour)

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Origin", "http://attacker.com")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code)
	if !hitHandler {
		t.Fatal("expected handler to be called even for disallowed origin")
	}
	assert.Equal(t, "", res.Header().Get("Access-Control-Allow-Origin"), "expected no CORS header for disallowed origin")
	assert.Equal(t, "", res.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORSMiddleware_PrefightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsMiddleware := middleware.NewCORSMiddleware(12 * time.Hour)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.OPTIONS("/api/v1/books/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/books/123", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "preflight OPTIONS should return 204")
	assert.Equal(t, "http://localhost:5173", res.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, PATCH", res.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsMiddleware := middleware.NewCORSMiddleware(12 * time.Hour)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, "", res.Header().Get("Access-Control-Allow-Origin"), "expected no CORS header when origin is not sent")
	assert.Equal(t, http.StatusOK, res.Code)
}

func TestCORSMiddleware_WildcardOriginMatching(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Default config allows only http://localhost:5173
	corsMiddleware := middleware.NewCORSMiddleware(12 * time.Hour)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	tests := []struct {
		origin  string
		allowed bool
	}{
		{"http://localhost:5173", true},
		{"http://attacker.com", false},
		{"http://localhost:3000", false},
		{"http://evil.localhost:5173", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
		req.Header.Set("Origin", tt.origin)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		if tt.allowed {
			assert.Equal(t, tt.origin, res.Header().Get("Access-Control-Allow-Origin"), "origin %s should be allowed", tt.origin)
		} else {
			assert.Equal(t, "", res.Header().Get("Access-Control-Allow-Origin"), "origin %s should not be allowed", tt.origin)
		}
	}
}

func TestCORSMiddleware_DefaultConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsMiddleware := middleware.NewCORSMiddleware(12 * time.Hour)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, "http://localhost:5173", res.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, res.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, res.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, res.Header().Get("Access-Control-Allow-Methods"), "DELETE")
	assert.Equal(t, "true", res.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_ExposedHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsMiddleware := middleware.NewCORSMiddleware(12 * time.Hour)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	exposedHeaders := res.Header().Get("Access-Control-Expose-Headers")
	assert.Contains(t, exposedHeaders, "X-New-Access-Token")
	assert.Contains(t, exposedHeaders, "X-New-Refresh-Token")
	assert.Contains(t, exposedHeaders, "X-User-ID")
	assert.Contains(t, exposedHeaders, "X-User-Role")
}

func TestCORSMiddleware_MaxAge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsMiddleware := middleware.NewCORSMiddleware(24 * time.Hour)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	maxAgeHeader := res.Header().Get("Access-Control-Max-Age")
	assert.Equal(t, "86400", maxAgeHeader, "expected max-age header to be 24 hours in seconds")
}

func TestCORSMiddleware_NoCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Default config has allowCredentials: true
	corsMiddleware := middleware.NewCORSMiddleware(12 * time.Hour)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, "true", res.Header().Get("Access-Control-Allow-Credentials"), "expected credentials header to be true for default config")
}

func TestCORSMiddleware_MultipleOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Default config allows only http://localhost:5173
	corsMiddleware := middleware.NewCORSMiddleware(12 * time.Hour)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	allowedOrigins := []string{"http://localhost:5173"}
	for _, origin := range allowedOrigins {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
		req.Header.Set("Origin", origin)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		assert.Equal(t, origin, res.Header().Get("Access-Control-Allow-Origin"), "expected origin %s to be allowed", origin)
	}

	disallowedOrigins := []string{"http://localhost:3000", "https://example.com", "http://attacker.com"}
	for _, origin := range disallowedOrigins {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
		req.Header.Set("Origin", origin)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		assert.Equal(t, "", res.Header().Get("Access-Control-Allow-Origin"), "expected origin %s to be blocked", origin)
	}
}
