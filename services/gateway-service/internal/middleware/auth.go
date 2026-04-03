package middleware

import (
	"strings"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/clients/user_service_client"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/errors"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type JWTHandlerClient = user_service_client.JWTRefresherClient

type JWTClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	jwt.StandardClaims
}

type AuthMiddleware struct {
	jwtSecret    []byte
	jwtAlgorithm string
	expMins      time.Duration
	skipPaths    map[string]bool

	jwtHandlerClient JWTHandlerClient

	logger *logger.Logger
}

func NewAuthMiddleware(jwtSecret []byte, jwtAlgorithm string, expMins time.Duration, jwtHandlerClient JWTHandlerClient, logger *logger.Logger) *AuthMiddleware {
	skipPaths := map[string]bool{
		"/api/v1/auth/register": true,
		"/api/v1/auth/login":    true,
		"/healthy":              true,
		"/ready":                true,
		"/metrics":              true,
	}

	if jwtAlgorithm == "" {
		jwtAlgorithm = "HS256"
	}

	return &AuthMiddleware{
		jwtSecret:        jwtSecret,
		jwtAlgorithm:     jwtAlgorithm,
		expMins:          expMins,
		skipPaths:        skipPaths,
		logger:           logger,
		jwtHandlerClient: jwtHandlerClient,
	}
}

func (m *AuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		authHeader := c.GetHeader("Authorization")

		// Check Skip Paths or Public Books
		if m.skipPaths[path] || strings.HasPrefix(c.Request.URL.Path, "/api/v1/books") {
			m.setGuestContext(c)
			c.Next()
			return
		}

		// Check Authorization Header
		if authHeader == "" {
			m.abortUnauthorized(c, "Missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			m.abortUnauthorized(c, "Invalid authorization header format")
			return
		}

		// Validate Token
		tokenStr := parts[1]
		claims, err := m.validateToken(tokenStr)
		if err != nil {
			m.abortUnauthorized(c, "Invalid or expired token")
			return
		}

		// Check Sliding Window for Token Refresh
		if m.isInSlidingWindow(claims.ExpiresAt) {
			m.logger.Info("Token is in sliding window, consider refreshing", logger.Fields{
				"user_id":    claims.UserID,
				"expires_at": time.Unix(claims.ExpiresAt, 0).Format(time.RFC3339),
			})

			rt := c.GetHeader("X-Refresh-Token")
			if rt != "" {
				newTokens, err := m.jwtHandlerClient.RefreshToken(c.Request.Context(), rt)
				if err == nil {
					c.Header("X-New-Access-Token", newTokens.AccessToken)
					c.Header("X-New-Refresh-Token", newTokens.RefreshToken)
					m.logger.Info("Token refreshed successfully", logger.Fields{
						"user_id": claims.UserID,
					})
				}
			}
		}

		c.Set("X-User-ID", claims.UserID)
		c.Set("X-User-Role", strings.ToUpper(claims.Role))
		c.Next()
	}
}

func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("X-User-Role")
		if role != "ADMIN" {
			m.abortForbidden(c, "Admin privileges required")
			return
		}
	}
}

func (m *AuthMiddleware) RequireAdminOrManager() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("X-User-Role")
		if role != "ADMIN" && role != "MANAGER" {
			m.abortForbidden(c, "Admin or Manager privileges required")
			return
		}
	}
}

/* HELPER METHODS */
func (m *AuthMiddleware) validateToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != m.jwtAlgorithm {
			return nil, errors.ErrUnauthorized.Clone().WithMessage("Unexpected signing method: " + token.Method.Alg())
		}

		return m.jwtSecret, nil
	})

	if err != nil {
		return nil, errors.WrapError(err, errors.CodeInternalError, "unexpected error during token validation")
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.ErrUnauthorized.Clone().WithMessage("Invalid token")
}

func (m *AuthMiddleware) setGuestContext(c *gin.Context) {
	c.Request.Header.Set("X-User-ID", uuid.New().String())
	c.Request.Header.Set("X-User-Role", "guest")
}

func (m *AuthMiddleware) abortUnauthorized(c *gin.Context, msg string) {
	appErr := errors.ErrUnauthorized.Clone().WithMessage(msg)
	c.AbortWithStatusJSON(int(appErr.HTTPStatus), appErr)
}

func (m *AuthMiddleware) abortForbidden(c *gin.Context, msg string) {
	appErr := errors.ErrForbidden.Clone().WithMessage(msg)
	c.AbortWithStatusJSON(int(appErr.HTTPStatus), appErr)
}

func (m *AuthMiddleware) isInSlidingWindow(expiresAt int64) bool {
	expTime := time.Unix(expiresAt, 0)
	timeLeft := time.Until(expTime)
	windowDuration := m.expMins / 5 // refresh within the last 20% of token lifetime
	if windowDuration < time.Minute {
		windowDuration = time.Minute
	}
	return timeLeft > 0 && timeLeft <= windowDuration
}
