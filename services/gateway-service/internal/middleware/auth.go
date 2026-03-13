package middleware

import (
	"strings"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/errors"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
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
}

func NewAuthMiddleware(jwtSecret []byte, jwtAlgorithm string, logger *logger.Logger) *AuthMiddleware {
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
		jwtSecret:    jwtSecret,
		jwtAlgorithm: jwtAlgorithm,
		logger:       logger,
		skipPaths:    skipPaths,
	}
}

func (m *AuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.skipPaths[c.FullPath()] {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			appErr := errors.ErrUnauthorized.Clone().WithMessage("Missing authorization header")
			m.logger.Warn("Missing authorization header")
			c.AbortWithStatusJSON(int(appErr.HTTPStatus), appErr)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
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

		c.Set("X-User-ID", claims.UserID)
		c.Set("X-User-Role", claims.Role)
	}
}

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

/* GLOBAL METHODS */
func GenerateToken(userID, role, email string, jwtSecret []byte, jwtAlgorithm string, expirationMins int) (string, error) {
	expirationTime := time.Now().Add(time.Duration(expirationMins) * time.Minute).Unix()
	issuedAt := time.Now().Unix()

	claims := &JWTClaims{
		UserID: userID,
		Role:   role,
		Email:  email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime,
			IssuedAt:  issuedAt,
			Issuer:    "lib-management-system",
			Subject:   userID,
			Audience:  "gateway-service",
		},
	}

	var signingMethod jwt.SigningMethod
	switch jwtAlgorithm {
	case "HS256":
		signingMethod = jwt.SigningMethodHS256
	case "HS512":
		signingMethod = jwt.SigningMethodHS512
	default:
		signingMethod = jwt.SigningMethodHS256
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", errors.WrapError(err, errors.CodeInternalError, "failed to sign token")
	}
	return tokenString, nil
}

func RefreshToken(tokenStr string, jwtSecret []byte, jwtAlgorithm string, expirationMins int) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwtAlgorithm {
			return nil, errors.ErrUnauthorized.Clone().WithMessage("Unexpected signing method: " + token.Method.Alg())
		}

		return jwtSecret, nil
	})

	if err != nil {
		return "", errors.WrapError(err, errors.CodeInternalError, "unexpected error during token validation")
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return GenerateToken(claims.UserID, claims.Role, claims.Email, jwtSecret, jwtAlgorithm, expirationMins)
	}

	return "", errors.ErrUnauthorized.Clone().WithMessage("Invalid token")
}
