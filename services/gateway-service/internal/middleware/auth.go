package middleware

import (
	"strings"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/clients"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/errors"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	jwt.StandardClaims
}

type AuthMiddleware struct {
	jwtSecret    []byte
	jwtAlgorithm string
	logger       *logger.Logger
	skipPaths    map[string]bool
	grpcHandler  *clients.ClientManager
}

func NewAuthMiddleware(jwtSecret []byte, jwtAlgorithm string, logger *logger.Logger, grpcHandler *clients.ClientManager) *AuthMiddleware {
	skipPaths := map[string]bool{
		"/api/v1/auth/register": true,
		"/api/v1/auth/login":    true,
		"/healthy":              true,
		"/ready":                true,
		"/metrics":              true,

		"/api/v1/books": true, // Allow public access to book listings
	}

	if jwtAlgorithm == "" {
		jwtAlgorithm = "HS256"
	}

	return &AuthMiddleware{
		jwtSecret:    jwtSecret,
		jwtAlgorithm: jwtAlgorithm,
		logger:       logger,
		skipPaths:    skipPaths,
		grpcHandler:  grpcHandler,
	}
}

func (m *AuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if m.skipPaths[c.FullPath()] || authHeader == "" {
			m.defaultHeader(c)
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			if m.skipPaths[c.FullPath()] {
				m.defaultHeader(c)
				c.Next()
				return
			}
			appErr := errors.ErrUnauthorized.Clone().WithMessage("Invalid authorization header format")
			m.logger.Warn("Invalid authorization header format")
			c.AbortWithStatusJSON(int(appErr.HTTPStatus), appErr)
			return
		}

		tokenStr := parts[1]
		claims, err := m.validateToken(tokenStr)
		if err != nil {
			appErr := errors.ErrUnauthorized.Clone().WithMessage("Invalid token attempt")
			m.logger.Warn("Invalid token attempt")
			c.AbortWithStatusJSON(int(appErr.HTTPStatus), appErr)
			return
		}

		if m.isInSlidingWindow(claims.ExpiresAt) {
			m.logger.Info("Token is in sliding window, consider refreshing", logger.Fields{
				"user_id":    claims.UserID,
				"expires_at": time.Unix(claims.ExpiresAt, 0).Format(time.RFC3339),
			})

			rt := c.GetHeader("X-Refresh-Token")
			newTokens, err := m.grpcHandler.RefreshToken(c, claims.UserID, rt)
			if err == nil {
				m.logger.Info("Token refreshed successfully", logger.Fields{
					"user_id": claims.UserID,
				})
				c.Header("X-New-Access-Token", newTokens.AccessToken)
				c.Header("X-New-Refresh-Token", newTokens.RefreshToken)
			}
		}

		c.Set("X-User-ID", claims.UserID)
		c.Set("X-User-Role", claims.Role)
		c.Next()
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

func (m *AuthMiddleware) defaultHeader(c *gin.Context) {
	c.Request.Header.Set("X-User-ID", uuid.New().String())
	c.Request.Header.Set("X-User-Role", "guest")
}

func (m *AuthMiddleware) isInSlidingWindow(expiresAt int64) bool {
	expTime := time.Unix(expiresAt, 0)
	timeLeft := time.Until(expTime)
	const windowDuration = 5 * time.Minute
	return timeLeft > 0 && timeLeft <= windowDuration
}
