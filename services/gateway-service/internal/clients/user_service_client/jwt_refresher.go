package user_service_client

import (
	"context"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/user_token_dto"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
)

type JWTRefresher struct {
	userServiceClient *UserServiceClient
	logger            *logger.Logger
}

func NewJWTRefresher(userServiceClient *UserServiceClient, logger *logger.Logger) *JWTRefresher {
	return &JWTRefresher{
		userServiceClient: userServiceClient,
		logger:            logger,
	}
}

func (jr *JWTRefresher) RefreshToken(ctx context.Context, refreshToken string) (user_token_dto.TokenPairDTO, error) {
	jr.logger.Info("Refreshing JWT token", logger.Fields{
		"user_id":       ctx.Value("user_id"),
		"refresh_token": refreshToken,
	})

	req := &userv1.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	resp, err := jr.userServiceClient.RefreshToken(ctx, req)
	if err != nil {
		jr.logger.Error("Failed to refresh JWT token", err, logger.Fields{
			"user_id": ctx.Value("user_id"),
		})
		return user_token_dto.TokenPairDTO{}, err
	}

	return user_token_dto.TokenPairDTO{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}, nil
}
