package applications

import (
	"crypto/sha256"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/errors"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/models"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	jwt.StandardClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type JWTService struct {
	jwtSecret    []byte
	jwtAlgorithm string
	jwtExpMins   time.Duration
}

func NewJWTService(jwtSecret []byte, jwtAlgorithm string, jwtExpMins time.Duration) *JWTService {
	return &JWTService{
		jwtSecret:    jwtSecret,
		jwtAlgorithm: jwtAlgorithm,
		jwtExpMins:   jwtExpMins,
	}
}

func (s *JWTService) GenerateTokenPair(user *models.User) (*TokenPair, string, time.Time, error) {
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	refreshToken := uuid.New().String() + "-" + uuid.New().String()
	refreshTokenHashed := s.HashRefreshToken(refreshToken)
	timeExpiresAt := time.Now().Add(time.Duration(s.jwtExpMins) * time.Minute)
	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, refreshTokenHashed, timeExpiresAt, nil

}

func (s *JWTService) generateAccessToken(user *models.User) (string, error) {
	expMins := int(s.jwtExpMins.Minutes())
	if expMins <= 0 {
		expMins = 60
	}

	claims := JWTClaims{
		UserID: user.ID,
		Role:   string(user.Role),
		Email:  user.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(expMins) * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "lib-management-system",
			Subject:   user.ID,
			Audience:  "user-service",
		},
	}

	var signingMethod jwt.SigningMethod
	switch s.jwtAlgorithm {
	case "HS512":
		signingMethod = jwt.SigningMethodHS512
	default:
		signingMethod = jwt.SigningMethodHS256
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	return token.SignedString(s.jwtSecret)

}

func (s *JWTService) HashRefreshToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return string(h.Sum(nil))
}

func (s *JWTService) ValidateAccessToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != s.jwtAlgorithm {
			return nil, errors.ErrUnauthorized.Clone().WithMessage("Unexpected signing method: " + token.Method.Alg())
		}

		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, errors.WrapError(err, errors.CodeInternalError, "unexpected error during token validation")
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.ErrUnauthorized.Clone().WithMessage("Invalid token")
}
