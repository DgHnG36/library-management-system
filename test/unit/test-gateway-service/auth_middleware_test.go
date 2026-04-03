package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/user_token_dto"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/middleware"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

/* STATIC ENV TESTING */

var jwtSecret = "local-lms-secret-key"
var jwtExpMins = 5 * time.Minute
var testLogger = logger.DefaultNewLogger()

/* MOCK JWT SERVICE */

type Mock_JWTService struct {
	mock.Mock
}

func (m *Mock_JWTService) RefreshToken(ctx context.Context, refreshToken string) (user_token_dto.TokenPairDTO, error) {
	args := m.Called(ctx, refreshToken)
	return args.Get(0).(user_token_dto.TokenPairDTO), args.Error(1)
}

/* HELPER METHODS */
func generateToken(userID, role, email string, secret []byte, algorithm string, expMins int) (string, error) {
	claims := middleware.JWTClaims{
		UserID: userID,
		Role:   role,
		Email:  email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(expMins) * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "lib-management-system",
			Subject:   userID,
		},
	}

	var signingMethod jwt.SigningMethod
	switch algorithm {
	case "HS512":
		signingMethod = jwt.SigningMethodHS512
	default:
		signingMethod = jwt.SigningMethodHS256
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	return token.SignedString(secret)
}

func refreshToken(tokenStr string, secret []byte, algorithm string, expMins int) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*middleware.JWTClaims)
	if !ok || !token.Valid {
		return "", jwt.ErrSignatureInvalid
	}

	return generateToken(claims.UserID, claims.Role, claims.Email, secret, algorithm, expMins)
}

func generateTokenWithDuration(userID, role, email string, secret []byte, algorithm string, d time.Duration) (string, error) {
	claims := middleware.JWTClaims{
		UserID: userID,
		Role:   role,
		Email:  email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(d).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "lib-management-system",
			Subject:   userID,
		},
	}

	var signingMethod jwt.SigningMethod
	switch algorithm {
	case "HS512":
		signingMethod = jwt.SigningMethodHS512
	default:
		signingMethod = jwt.SigningMethodHS256
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	return token.SignedString(secret)
}

/* TESTCASE */
func TestAuthMiddleware_SkipPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, nil, nil)

	tests := []struct {
		name string
		path string
	}{
		{"Healthy path", "/healthy"},
		{"Ready path", "/ready"},
		{"Metrics path", "/metrics"},
		{"Register path", "/api/v1/auth/register"},
		{"Login path", "/api/v1/auth/login"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			hitHandler := false

			router.GET(tt.path, mw.Handle(), func(c *gin.Context) {
				hitHandler = true
				c.Status(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			assert.Equal(t, http.StatusNoContent, res.Code, "Path %s should be skipped", tt.path)
			if !hitHandler {
				t.Errorf("expected downstream handler to be called for skip path: %s", tt.path)
			}
		})
	}
}

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, &Mock_JWTService{}, nil)

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

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, &Mock_JWTService{}, nil)

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Authorization", "Invalid-Format-Token")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusUnauthorized, res.Code, "expected status %d, got %d", http.StatusUnauthorized, res.Code)

	if hitHandler {
		t.Fatal("expected downstream handler not to be called when auth header has invalid format")
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, &Mock_JWTService{}, nil)

	router := gin.New()
	hitHandler := false
	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer Invalid-Token-String")
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

	userID := "user-test-valid-token"
	userRole := "admin"
	userEmail := "user-test-valid-token@test.com"
	token, err := generateToken(userID, userRole, userEmail, []byte(jwtSecret), "HS256", int(jwtExpMins.Minutes()))
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	router := gin.New()
	hitHandler := false
	userIDRes := ""
	userRoleRes := ""
	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		if uid, exists := c.Get("X-User-ID"); exists {
			userIDRes = uid.(string)
		}
		if role, exists := c.Get("X-User-Role"); exists {
			userRoleRes = role.(string)
		}
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
	assert.Equal(t, "user-test-valid-token", userIDRes, "expected user id %q, got %v", "user-test-valid-token", userIDRes)
	assert.Equal(t, "ADMIN", userRoleRes, "expected user role %q, got %v", "ADMIN", userRoleRes)
}

func TestAuthMiddleware_SlidingWindow_ExpiredAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newTokenPair := user_token_dto.TokenPairDTO{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
	}
	mockSvc := &Mock_JWTService{}
	mockSvc.On("RefreshToken", mock.Anything, "test-refresh-token").Return(newTokenPair, nil)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, mockSvc, testLogger)

	// Token expiring in 30s — within sliding window (jwtExpMins/5 = 1min)
	token, err := generateTokenWithDuration("user-sliding", "admin", "user-sliding@test.com", []byte(jwtSecret), "HS256", 30*time.Second)
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
	req.Header.Set("X-Refresh-Token", "test-refresh-token")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "expected status %d, got %d", http.StatusNoContent, res.Code)
	if !hitHandler {
		t.Fatal("expected downstream handler to be called for valid token in sliding window")
	}
	assert.Equal(t, "new-access-token", res.Header().Get("X-New-Access-Token"), "expected new access token in response header")
	assert.Equal(t, "new-refresh-token", res.Header().Get("X-New-Refresh-Token"), "expected new refresh token in response header")
	mockSvc.AssertExpectations(t)
}

func TestAuthMiddleware_JWTAlgorithm_HS512(t *testing.T) {
	token, err := generateToken("user-change-algorithm", "admin", "user-change-algorithm@test.com", []byte(jwtSecret), "HS512", int(jwtExpMins.Minutes()))
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS512", jwtExpMins, nil, testLogger)
	router := gin.New()
	hitHandler := false
	var userIDRes, userRoleRes string

	router.GET("/api/v1/orders", mw.Handle(), func(c *gin.Context) {
		hitHandler = true
		if uid, exists := c.Get("X-User-ID"); exists {
			userIDRes = uid.(string)
		}
		if role, exists := c.Get("X-User-Role"); exists {
			userRoleRes = role.(string)
		}
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

	assert.Equal(t, "user-change-algorithm", userIDRes, "expected user id %q from HS512 token, got %v", "user-change-algorithm", userIDRes)
	assert.Equal(t, "ADMIN", userRoleRes, "expected user role %q from HS512 token, got %v", "ADMIN", userRoleRes)
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	wrongSecret := []byte("LoCal-Lms-SeCret-KeY")

	token, err := generateToken("user-wrong-secret", "admin", "user-wrong-secret@test.com", []byte(jwtSecret), "HS256", 10)
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

func TestAuthMiddleware_AlgorithmMisMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := generateToken("user-algorithm-miss-match", "admin", "user-algorithm-miss-match@test.com", []byte(jwtSecret), "HS512", 10)
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

func TestAuthMiddleware_RefreshToken_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newTokenPair := user_token_dto.TokenPairDTO{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
	}
	mockSvc := &Mock_JWTService{}
	mockSvc.On("RefreshToken", mock.Anything, "valid-refresh-token").Return(newTokenPair, nil)

	mw := middleware.NewAuthMiddleware([]byte(jwtSecret), "HS256", jwtExpMins, mockSvc, testLogger)

	token, err := generateTokenWithDuration("user-refresh-token", "admin", "user-refresh-token@test.com", []byte(jwtSecret), "HS256", 30*time.Second)
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
	req.Header.Set("X-Refresh-Token", "valid-refresh-token")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code, "expected status %d for refreshed token, got %d", http.StatusNoContent, res.Code)
	if !hitHandler {
		t.Fatal("expected downstream handler to be called for refreshed token")
	}
	assert.Equal(t, "user-refresh-token", userID, "expected user id %q from refreshed token, got %v", "user-refresh-token", userID)
	assert.Equal(t, "ADMIN", userRole, "expected user role %q from refreshed token, got %v", "ADMIN", userRole)
	assert.Equal(t, "new-access-token", res.Header().Get("X-New-Access-Token"))
	assert.Equal(t, "new-refresh-token", res.Header().Get("X-New-Refresh-Token"))
	mockSvc.AssertExpectations(t)
}
