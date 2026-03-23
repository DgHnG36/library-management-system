package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/middleware"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var jwtSecret = "secret-jwt-key"

func TestAuthMiddleware_SkipPathHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", nil)

	router := gin.New()
	hitHandler := false
	router.GET("/healthy", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/healthy", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "expected status %d, got %d", http.StatusNoContent, res.Code)

	if !hitHandler {
		t.Fatal("expected downstream handler to be called for skip path")
	}
}

func TestAuthMiddleware_SkipPathReady(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", nil)

	router := gin.New()
	hitHandler := false
	router.GET("/ready", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "expected status %d, got %d", http.StatusNoContent, res.Code)

	if !hitHandler {
		t.Fatal("expected downstream handler to be called for skip path")
	}
}

func TestAuthMiddleware_SkipPathMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", nil)

	router := gin.New()
	hitHandler := false

	router.GET("/metrics", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "expected status %d, got %d", http.StatusNoContent, res.Code)

	if !hitHandler {
		t.Fatal("expected downstream handler to be called for skip path")
	}
}

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", logger.DefaultNewLogger())

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/books", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code, "expected status %d, got %d", http.StatusUnauthorized, res.Code)

	if hitHandler {
		t.Fatal("expected downstream handler not to be called when auth header is missing")
	}
}

func TestAuthMiddleware_InvalidAuthorizationFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", logger.DefaultNewLogger())

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/books", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Authorization", "InvalidFormatToken")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code, "expected status %d, got %d", http.StatusUnauthorized, res.Code)

	if hitHandler {
		t.Fatal("expected downstream handler not to be called when auth header has invalid format")
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", logger.DefaultNewLogger())

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/books", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Authorization", "Bearer InvalidTokenString")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code, "expected status %d, got %d", http.StatusUnauthorized, res.Code)

	if hitHandler {
		t.Fatal("expected downstream handler not to be called when token is invalid")
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", logger.DefaultNewLogger())
	token, err := middleware.GenerateToken("user-test", "admin", "user@test.com", []byte(jwtSecret), "HS256", 15)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	router := gin.New()
	hitHandler := false
	var userID any
	var userRole any
	router.GET("/api/v1/books", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		userID, _ = c.Get("X-User-ID")
		userRole, _ = c.Get("X-User-Role")
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "expected status %d, got %d", http.StatusNoContent, res.Code)

	if !hitHandler {
		t.Fatal("expected downstream handler to be called for valid token")
	}
	assert.Equal(t, "user-test", userID, "expected user id %q, got %v", "user-test", userID)
	assert.Equal(t, "admin", userRole, "expected user role %q, got %v", "admin", userRole)
}

func TestAuthMiddleware_SkipPathWithValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", logger.DefaultNewLogger())
	token, err := middleware.GenerateToken("user-test", "admin", "user@test.com", []byte(jwtSecret), "HS256", 15)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/books", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "expected status %d, got %d", http.StatusNoContent, res.Code)

	if !hitHandler {
		t.Fatal("expected downstream handler to be called for valid token")
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", logger.DefaultNewLogger())
	token, err := middleware.GenerateToken("user-test", "admin", "user@test.com", []byte(jwtSecret), "HS256", -1) // Expire immediately
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/books", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code, "expected status %d, got %d", http.StatusUnauthorized, res.Code)

	if hitHandler {
		t.Fatal("expected downstream handler not to be called when token is expired")
	}
}

func TestAuthMiddleware_SkipPathAuthRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", logger.DefaultNewLogger())

	router := gin.New()
	hitHandler := false
	router.POST("/api/v1/auth/register", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "expected status %d, got %d", http.StatusNoContent, res.Code)
	if !hitHandler {
		t.Fatal("expected downstream handler to be called for /api/v1/auth/register skip path")
	}
}

func TestAuthMiddleware_SkipPathAuthLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", logger.DefaultNewLogger())

	router := gin.New()
	hitHandler := false
	router.POST("/api/v1/auth/login", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "expected status %d, got %d", http.StatusNoContent, res.Code)
	if !hitHandler {
		t.Fatal("expected downstream handler to be called for /api/v1/auth/login skip path")
	}
}

func TestGenerateToken_HS512(t *testing.T) {
	token, err := middleware.GenerateToken("user-123", "admin", "user@test.com", []byte(jwtSecret), "HS512", 15)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS512", logger.DefaultNewLogger())
	router := gin.New()
	hitHandler := false

	router.GET("/api/v1/books", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "expected status %d for HS512 token, got %d", http.StatusNoContent, res.Code)
	if !hitHandler {
		t.Fatal("expected downstream handler to be called for valid HS512 token")
	}
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	correctSecret := []byte("correct-secret")
	wrongSecret := []byte("wrong-secret")

	token, err := middleware.GenerateToken("user-123", "admin", "user@test.com", correctSecret, "HS256", 15)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	mw := middleware.NewAuthMiddleware(wrongSecret, "HS256", logger.DefaultNewLogger())
	router := gin.New()
	hitHandler := false

	router.GET("/api/v1/books", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code, "expected status %d for token with wrong secret, got %d", http.StatusUnauthorized, res.Code)
	if hitHandler {
		t.Fatal("expected downstream handler not to be called for token with wrong secret")
	}
}

func TestAuthMiddleware_AlgorithmMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := middleware.GenerateToken("user-123", "admin", "user@test.com", []byte(jwtSecret), "HS512", 15)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", logger.DefaultNewLogger())
	router := gin.New()
	hitHandler := false

	router.GET("/api/v1/books", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code, "expected status %d for algorithm mismatch, got %d", http.StatusUnauthorized, res.Code)
	if hitHandler {
		t.Fatal("expected downstream handler not to be called when token algorithm doesn't match middleware")
	}
}

func TestRefreshToken_ValidToken(t *testing.T) {
	originalToken, err := middleware.GenerateToken("user-123", "admin", "user@test.com", []byte(jwtSecret), "HS256", 15)
	if err != nil {
		t.Fatalf("failed to generate original token: %v", err)
	}

	refreshedToken, err := middleware.RefreshToken(originalToken, []byte(jwtSecret), "HS256", 30)
	if err != nil {
		t.Fatalf("failed to refresh token: %v", err)
	}

	if refreshedToken == "" {
		t.Fatal("expected non-empty refreshed token")
	}

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", logger.DefaultNewLogger())
	router := gin.New()
	hitHandler := false
	var userID any
	var userRole any

	router.GET("/api/v1/books", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		userID, _ = c.Get("X-User-ID")
		userRole, _ = c.Get("X-User-Role")
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books", nil)
	req.Header.Set("Authorization", "Bearer "+refreshedToken)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "expected status %d for refreshed token, got %d", http.StatusNoContent, res.Code)
	if !hitHandler {
		t.Fatal("expected downstream handler to be called for refreshed token")
	}
	assert.Equal(t, "user-123", userID, "expected user id %q from refreshed token, got %v", "user-123", userID)
	assert.Equal(t, "admin", userRole, "expected user role %q from refreshed token, got %v", "admin", userRole)
}
