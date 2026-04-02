package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/middleware"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* STATIC ENV TESTING */

var jwtSecret = "local-lms-secret-key"
var jwtExpMins = 5 * time.Minute
var testLogger = logger.DefaultNewLogger()

/* MOCK JWT SERVICE */
type Mock_JWTClaims struct {
	MockUserID string `json:"mock_user_id"`
	MockRole   string `json:"mock_role"`
	MockEmail  string `json:"mock_email"`
	jwt.StandardClaims
}

type Mock_TokenPair struct {
	MockAccessToken  string `json:"mock_access_token"`
	MockRefreshToken string `json:"mock_refresh_token"`
}

type Mock_JWTService struct {
	mockJWTSecret    []byte
	mockJWTAlgorithm string
	mockExpMins      time.Duration
}

func NewMock_JWTService(mockJWTSecret []byte, mockJWTAlgorithm string, mockExpMins time.Duration) *Mock_JWTService {
	return &Mock_JWTService{
		mockJWTSecret:    mockJWTSecret,
		mockJWTAlgorithm: mockJWTAlgorithm,
		mockExpMins:      mockExpMins,
	}
}

func (m *Mock_JWTService) MockGenerateTokenPair(mockUserID, mockRole, mockEmail string) (*Mock_TokenPair, string, time.Time, error) {
	mockAccessToken, err := m.mockGenerateAccessToken(mockUserID, mockRole, mockEmail)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	mockRefreshToken := uuid.New().String() + "-" + uuid.New().String()
	mockRefreshTokenHashed := m.HashMockRefreshToken(mockRefreshToken)
	timeExpiresAt := time.Now().Add(time.Duration(m.mockExpMins) * time.Minute)
	return &Mock_TokenPair{
		MockAccessToken:  mockAccessToken,
		MockRefreshToken: mockRefreshToken,
	}, mockRefreshTokenHashed, timeExpiresAt, nil
}

func (m *Mock_JWTService) mockGenerateAccessToken(mockUserID, mockRole, mockEmail string) (string, error) {
	expMins := int(m.mockExpMins.Minutes())
	if expMins <= 0 {
		expMins = 60
	}

	claims := Mock_JWTClaims{
		MockUserID: mockUserID,
		MockRole:   mockRole,
		MockEmail:  mockEmail,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(expMins) * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "lib-management-system-test",
			Subject:   mockUserID,
			Audience:  "user-service-test",
		},
	}

	var signingMethod jwt.SigningMethod
	switch m.mockJWTAlgorithm {
	case "HS256":
		signingMethod = jwt.SigningMethodHS256
	case "HS512":
		signingMethod = jwt.SigningMethodHS512
	default:
		signingMethod = jwt.SigningMethodHS256
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	signedToken, err := token.SignedString(m.mockJWTSecret)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func (m *Mock_JWTService) HashMockRefreshToken(refreshToken string) string {
	// For testing purposes, we can just return a simple hash (in production, use a secure hash function)
	return "hashed-" + refreshToken
}

func (m *Mock_JWTService) MockValidateAccessToken(tokenStr string) (*Mock_JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Mock_JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != m.mockJWTAlgorithm {
			return nil, status.Error(codes.Unauthenticated, "unexpected signing method")
		}

		return m.mockJWTSecret, nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, "unexpected error during token validation")
	}

	if claims, ok := token.Claims.(*Mock_JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, status.Error(codes.Unauthenticated, "invalid token")
}

/* TESTCASE */
func TestAuthMiddleware_SkipPathHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, nil)

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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, nil)

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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, nil)

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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, nil)

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code, "expected status %d, got %d", http.StatusUnauthorized, res.Code)

	if hitHandler {
		t.Fatal("expected downstream handler not to be called when auth header is missing")
	}
}

func TestAuthMiddleware_InvalidAuthorizationFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, nil)

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, nil)

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, testLogger)
	token, err := middleware.GenerateToken("user-test", "admin", "user@test.com", []byte(jwtSecret), "HS256", 15)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	router := gin.New()
	hitHandler := false
	var userID any
	var userRole any
	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		userID, _ = c.Get("X-User-ID")
		userRole, _ = c.Get("X-User-Role")
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, testLogger)
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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, testLogger)
	token, err := middleware.GenerateToken("user-test", "admin", "user@test.com", []byte(jwtSecret), "HS256", -1) // Expire immediately
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, testLogger)

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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, testLogger)

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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS512", jwtExpMins, nil, testLogger)
	router := gin.New()
	hitHandler := false

	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
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

	mw := middleware.NewAuthMiddleware(wrongSecret, "HS256", jwtExpMins, nil, testLogger)
	router := gin.New()
	hitHandler := false

	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, testLogger)
	router := gin.New()
	hitHandler := false

	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, testLogger)
	router := gin.New()
	hitHandler := false
	var userID any
	var userRole any

	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		userID, _ = c.Get("X-User-ID")
		userRole, _ = c.Get("X-User-Role")
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
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
