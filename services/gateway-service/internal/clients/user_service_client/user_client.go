package user_service_client

import (
	"context"
	"fmt"

	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// UserServiceClient wraps the generated UserServiceClient with additional functionality
type UserServiceClient struct {
	client userv1.UserServiceClient
	conn   *grpc.ClientConn
	logger *logger.Logger
}

// NewUserServiceClient creates a new user service client connection
func NewUserServiceClient(addr string, logger *logger.Logger) (*UserServiceClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*100)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service at %s: %w", addr, err)
	}

	return &UserServiceClient{
		client: userv1.NewUserServiceClient(conn),
		conn:   conn,
		logger: logger,
	}, nil
}

// Register registers a new user
func (uc *UserServiceClient) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	uc.logger.Info("Register called", map[string]interface{}{
		"username": req.Username,
		"email":    req.Email,
	})

	resp, err := uc.client.Register(ctx, req)
	if err != nil {
		uc.logger.Error("Register failed", err, map[string]interface{}{
			"username": req.Username,
			"email":    req.Email,
		})
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	return resp, nil
}

// Login authenticates a user and returns tokens
func (uc *UserServiceClient) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	uc.logger.Info("Login called", map[string]interface{}{
		"username": req.Username,
	})

	resp, err := uc.client.Login(ctx, req)
	if err != nil {
		uc.logger.Error("Login failed", err, map[string]interface{}{
			"username": req.Username,
		})
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	return resp, nil
}

// GetProfile retrieves a user's profile information
func (uc *UserServiceClient) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.UserProfileResponse, error) {
	uc.logger.Info("GetProfile called", map[string]interface{}{
		"user_id": req.UserId,
	})

	resp, err := uc.client.GetProfile(ctx, req)
	if err != nil {
		uc.logger.Error("GetProfile failed", err, map[string]interface{}{
			"user_id": req.UserId,
		})
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return resp, nil
}

// UpdateProfile updates a user's profile information
func (uc *UserServiceClient) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UserProfileResponse, error) {
	uc.logger.Info("UpdateProfile called", map[string]interface{}{
		"user_id": req.UserId,
	})

	resp, err := uc.client.UpdateProfile(ctx, req)
	if err != nil {
		uc.logger.Error("UpdateProfile failed", err, map[string]interface{}{
			"user_id": req.UserId,
		})
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return resp, nil
}

// UpdateVIPAccount upgrades a user to VIP status
func (uc *UserServiceClient) UpdateVIPAccount(ctx context.Context, req *userv1.UpdateVIPAccountRequest) (*userv1.UpdateVIPAccountResponse, error) {
	uc.logger.Info("UpdateVIPAccount called", map[string]interface{}{
		"user_id": req.UserId,
	})

	resp, err := uc.client.UpdateVIPAccount(ctx, req)
	if err != nil {
		uc.logger.Error("UpdateVIPAccount failed", err, map[string]interface{}{
			"user_id": req.UserId,
		})
		return nil, fmt.Errorf("failed to update VIP account: %w", err)
	}

	return resp, nil
}

// ListUsers retrieves a list of users with pagination
func (uc *UserServiceClient) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	uc.logger.Info("ListUsers called")

	resp, err := uc.client.ListUsers(ctx, req)
	if err != nil {
		uc.logger.Error("ListUsers failed", err)
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return resp, nil
}

// DeleteUsers deletes multiple users
func (uc *UserServiceClient) DeleteUsers(ctx context.Context, req *userv1.DeleteUsersRequest) error {
	uc.logger.Info("DeleteUsers called", map[string]interface{}{
		"ids": req.Ids,
	})

	_, err := uc.client.DeleteUsers(ctx, req)
	if err != nil {
		uc.logger.Error("DeleteUsers failed", err, map[string]interface{}{
			"ids": req.Ids,
		})
		return fmt.Errorf("failed to delete users: %w", err)
	}

	return nil
}

// RefreshToken refreshes an expired token
func (uc *UserServiceClient) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.LoginResponse, error) {
	uc.logger.Info("RefreshToken called", map[string]interface{}{
		"user_id": req.UserId,
	})

	resp, err := uc.client.RefreshToken(ctx, req)
	if err != nil {
		uc.logger.Error("RefreshToken failed", err, map[string]interface{}{
			"user_id": req.UserId,
		})
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return resp, nil
}

// Close closes the connection to the user service
func (uc *UserServiceClient) Close() error {
	if uc.conn != nil {
		return uc.conn.Close()
	}
	return nil
}

// GetConnection returns the underlying gRPC connection
func (uc *UserServiceClient) GetConnection() *grpc.ClientConn {
	return uc.conn
}

// GetClient returns the underlying generated client
func (uc *UserServiceClient) GetClient() userv1.UserServiceClient {
	return uc.client
}
