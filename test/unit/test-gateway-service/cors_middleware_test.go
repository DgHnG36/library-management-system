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
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", res.Header().Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, res.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "true", res.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsMiddleware := middleware.NewCORSMiddleware(12 * time.Hour)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Origin", "http://attacker.com")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, "", res.Header().Get("Access-Control-Allow-Origin"), "expected no CORS header for disallowed origin")
	assert.Equal(t, "", res.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, http.StatusOK, res.Code)
}

func TestCORSMiddleware_PrefightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsMiddleware := middleware.NewCORSMiddleware(12 * time.Hour)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	// Register an OPTIONS route so gin routes the preflight request through the middleware
	router.OPTIONS("/api/v1/books/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/books/123", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "preflight OPTIONS should return 204")
	assert.Equal(t, "http://localhost:5173", res.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", res.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsMiddleware := middleware.NewCORSMiddleware(
		[]string{"http://localhost:3000"},
		[]string{"GET", "POST", "OPTIONS"},
		[]string{"Content-Type", "Authorization"},
		[]string{"Content-Type"},
		true,
		12*time.Hour,
	)

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

	corsMiddleware := middleware.NewCORSMiddleware(
		[]string{"*.example.com"},
		[]string{"GET", "POST", "OPTIONS"},
		[]string{"Content-Type", "Authorization"},
		[]string{"Content-Type"},
		true,
		12*time.Hour,
	)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	tests := []struct {
		origin  string
		allowed bool
	}{
		{"http://app.example.com", true},
		{"http://api.example.com", true},
		{"http://example.com", false},
		{"http://app.other.com", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
		req.Header.Set("Origin", tt.origin)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		if tt.allowed {
			assert.Equal(t, tt.origin, res.Header().Get("Access-Control-Allow-Origin"), "origin %s should match *.example.com", tt.origin)
		} else {
			assert.Equal(t, "", res.Header().Get("Access-Control-Allow-Origin"), "origin %s should not match *.example.com", tt.origin)
		}
	}
}

func TestCORSMiddleware_DefaultConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsMiddleware := middleware.NewDefaultCORSMiddleware()

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, "http://localhost:3000", res.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, res.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, res.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, res.Header().Get("Access-Control-Allow-Methods"), "DELETE")
	assert.Equal(t, "true", res.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_ExposedHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsMiddleware := middleware.NewCORSMiddleware(
		[]string{"http://localhost:3000"},
		[]string{"GET", "POST", "OPTIONS"},
		[]string{"Content-Type", "Authorization"},
		[]string{"X-Custom-Header", "X-Request-ID"},
		false,
		12*time.Hour,
	)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.Header("X-Custom-Header", "custom-value")
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	exposedHeaders := res.Header().Get("Access-Control-Expose-Headers")
	assert.Contains(t, exposedHeaders, "X-Custom-Header")
	assert.Contains(t, exposedHeaders, "X-Request-ID")
}

func TestCORSMiddleware_MaxAge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	maxAge := 24 * time.Hour
	corsMiddleware := middleware.NewCORSMiddleware(
		[]string{"http://localhost:3000"},
		[]string{"GET", "POST", "OPTIONS"},
		[]string{"Content-Type", "Authorization"},
		[]string{"Content-Type"},
		true,
		maxAge,
	)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	maxAgeHeader := res.Header().Get("Access-Control-Max-Age")
	assert.Equal(t, "86400", maxAgeHeader, "expected max-age header to be 24 hours in seconds")
}

func TestCORSMiddleware_NoCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsMiddleware := middleware.NewCORSMiddleware(
		[]string{"http://localhost:3000"},
		[]string{"GET", "POST", "OPTIONS"},
		[]string{"Content-Type", "Authorization"},
		[]string{"Content-Type"},
		false,
		12*time.Hour,
	)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	credentialsHeader := res.Header().Get("Access-Control-Allow-Credentials")
	assert.Equal(t, "", credentialsHeader, "expected no credentials header when allowCredentials is false")
}

func TestCORSMiddleware_MultipleOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	corsMiddleware := middleware.NewCORSMiddleware(
		[]string{"http://localhost:3000", "http://localhost:5173", "https://example.com"},
		[]string{"GET", "POST", "OPTIONS"},
		[]string{"Content-Type", "Authorization"},
		[]string{"Content-Type"},
		true,
		12*time.Hour,
	)

	router := gin.New()
	router.Use(corsMiddleware.Handle())
	router.GET("/api/v1/books", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"books": "list"})
	})

	origins := []string{"http://localhost:3000", "http://localhost:5173", "https://example.com"}
	for _, origin := range origins {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
		req.Header.Set("Origin", origin)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		assert.Equal(t, origin, res.Header().Get("Access-Control-Allow-Origin"), "expected origin %s to be allowed", origin)
	}
}
