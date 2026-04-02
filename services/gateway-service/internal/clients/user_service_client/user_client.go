package user_service_client

import (
	"context"
	"fmt"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserServiceClient struct {
	client userv1.UserServiceClient
	conn   *grpc.ClientConn
	logger *logger.Logger
}

func NewUserServiceClient(addr string, log *logger.Logger) (*UserServiceClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(4*1024*1024)),
	)
	if err != nil {
		log.Error("Failed to connect to user service", err, logger.Fields{
			"address": addr,
		})
		return nil, fmt.Errorf("failed to connect to user service at %s: %w", addr, err)
	}

	log.Info("Successfully connected to user service", logger.Fields{
		"address": addr,
	})

	return &UserServiceClient{
		client: userv1.NewUserServiceClient(conn),
		conn:   conn,
		logger: log,
	}, nil
}

/* METHODS HANDLER */
func (uc *UserServiceClient) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	uc.logger.Info("Register called to user service", logger.Fields{
		"username":     req.GetUsername(),
		"email":        req.GetEmail(),
		"phone_number": req.GetPhoneNumber(),
	})

	resp, err := uc.client.Register(ctx, req)
	if err != nil {
		uc.logger.Error("Register failed", err, logger.Fields{
			"username":     req.GetUsername(),
			"email":        req.GetEmail(),
			"phone_number": req.GetPhoneNumber(),
		})
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	return resp, nil
}

func (uc *UserServiceClient) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	uc.logger.Info("Login called to user service", logger.Fields{
		"identifier": req.GetIdentifier(),
	})

	resp, err := uc.client.Login(ctx, req)
	if err != nil {
		uc.logger.Error("Login failed", err, logger.Fields{
			"identifier": req.Identifier,
		})
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	return resp, nil
}

func (uc *UserServiceClient) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.UserProfileResponse, error) {
	uc.logger.Info("GetProfile called to user service", logger.Fields{
		"user_id": req.GetId(),
	})

	resp, err := uc.client.GetProfile(ctx, req)
	if err != nil {
		uc.logger.Error("GetProfile failed", err, logger.Fields{
			"user_id": req.GetId(),
		})
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return resp, nil
}

func (uc *UserServiceClient) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UserProfileResponse, error) {
	uc.logger.Info("UpdateProfile called to user service", logger.Fields{
		"user_id": req.GetId(),
	})

	resp, err := uc.client.UpdateProfile(ctx, req)
	if err != nil {
		uc.logger.Error("UpdateProfile failed", err, logger.Fields{
			"user_id": req.GetId(),
		})
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return resp, nil
}

func (uc *UserServiceClient) UpdateVIPAccount(ctx context.Context, req *userv1.UpdateVIPAccountRequest) (*userv1.UpdateVIPAccountResponse, error) {
	uc.logger.Info("UpdateVIPAccount called to user service", logger.Fields{
		"user_id": req.GetId(),
	})

	resp, err := uc.client.UpdateVIPAccount(ctx, req)
	if err != nil {
		uc.logger.Error("UpdateVIPAccount failed", err, logger.Fields{
			"user_id": req.GetId(),
		})
		return nil, fmt.Errorf("failed to update VIP account: %w", err)
	}

	return resp, nil
}

func (uc *UserServiceClient) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	uc.logger.Info("ListUsers called to user service")

	resp, err := uc.client.ListUsers(ctx, req)
	if err != nil {
		uc.logger.Error("ListUsers failed", err)
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return resp, nil
}

func (uc *UserServiceClient) DeleteUsers(ctx context.Context, req *userv1.DeleteUsersRequest) error {
	uc.logger.Info("DeleteUsers called to user service", logger.Fields{
		"ids": req.GetIds(),
	})

	_, err := uc.client.DeleteUsers(ctx, req)
	if err != nil {
		uc.logger.Error("DeleteUsers failed", err, logger.Fields{
			"ids": req.GetIds(),
		})
		return fmt.Errorf("failed to delete users: %w", err)
	}

	return nil
}

func (uc *UserServiceClient) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.LoginResponse, error) {
	uc.logger.Info("RefreshToken called to user service", logger.Fields{
		"user_id":       ctx.Value("user_id"),
		"refresh_token": req.GetRefreshToken(),
	})

	resp, err := uc.client.RefreshToken(ctx, req)
	if err != nil {
		uc.logger.Error("RefreshToken failed", err, logger.Fields{
			"user_id":       ctx.Value("user_id"),
			"refresh_token": req.GetRefreshToken(),
		})
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return resp, nil
}

func (uc *UserServiceClient) Close() error {
	uc.logger.Info("Closing connection to user service")

	if uc.conn != nil {
		return uc.conn.Close()
	}
	return nil
}

func (uc *UserServiceClient) GetConnection() *grpc.ClientConn {
	return uc.conn
}
