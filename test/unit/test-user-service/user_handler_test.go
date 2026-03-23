package main

import (
	"context"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/user-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/handlers"
	"github.com/DgHnG36/lib-management-system/services/user-service/pkg/logger"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestUserHandler_Register_Success tests successful user registration
func TestUserHandler_Register_Success(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()
	req := &userv1.RegisterRequest{
		Username:    "testuser",
		Password:    "password123",
		Email:       "test@example.com",
		PhoneNumber: "+1234567890",
	}

	resp, err := handler.Register(ctx, req)

	assert.NoError(t, err, "expected no error")
	assert.NotNil(t, resp)
	assert.Equal(t, int32(200), resp.GetStatus())
	assert.Equal(t, "User registered successfully", resp.GetMessage())
	assert.NotEmpty(t, resp.GetUserId())
}

// TestUserHandler_Register_MissingUsername tests registration with missing username
func TestUserHandler_Register_MissingUsername(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()
	req := &userv1.RegisterRequest{
		Username: "",
		Password: "password123",
		Email:    "test@example.com",
	}

	resp, err := handler.Register(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestUserHandler_Register_MissingPassword tests registration with missing password
func TestUserHandler_Register_MissingPassword(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()
	req := &userv1.RegisterRequest{
		Username: "testuser",
		Password: "",
		Email:    "test@example.com",
	}

	resp, err := handler.Register(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestUserHandler_Register_MissingEmail tests registration with missing email
func TestUserHandler_Register_MissingEmail(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()
	req := &userv1.RegisterRequest{
		Username: "testuser",
		Password: "password123",
		Email:    "",
	}

	resp, err := handler.Register(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestUserHandler_Login_SuccessWithUsername tests successful login with username
func TestUserHandler_Login_SuccessWithUsername(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()

	// First register a user
	_, err := userSvc.Register(ctx, "testuser", "password123", "test@example.com", "+1234567890")
	assert.NoError(t, err)

	// Then login
	req := &userv1.LoginRequest{
		Password: "password123",
		Identifier: &userv1.LoginRequest_Username{
			Username: "testuser",
		},
	}

	resp, err := handler.Login(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(200), resp.GetStatus())
	assert.Equal(t, "Login successful", resp.GetMessage())
	assert.NotEmpty(t, resp.GetToken())
	assert.NotNil(t, resp.GetUser())
	assert.Equal(t, "testuser", resp.GetUser().GetUsername())
}

// TestUserHandler_Login_SuccessWithEmail tests successful login with email
func TestUserHandler_Login_SuccessWithEmail(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()

	// First register a user
	_, err := userSvc.Register(ctx, "testuser", "password123", "test@example.com", "+1234567890")
	assert.NoError(t, err)

	// Then login with email
	req := &userv1.LoginRequest{
		Password: "password123",
		Identifier: &userv1.LoginRequest_Email{
			Email: "test@example.com",
		},
	}

	resp, err := handler.Login(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(200), resp.GetStatus())
	assert.NotEmpty(t, resp.GetToken())
	assert.Equal(t, "test@example.com", resp.GetUser().GetEmail())
}

// TestUserHandler_Login_MissingPassword tests login with missing password
func TestUserHandler_Login_MissingPassword(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()
	req := &userv1.LoginRequest{
		Password: "",
		Identifier: &userv1.LoginRequest_Username{
			Username: "testuser",
		},
	}

	resp, err := handler.Login(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestUserHandler_Login_MissingIdentifier tests login with no username or email
func TestUserHandler_Login_MissingIdentifier(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()
	req := &userv1.LoginRequest{
		Password:   "password123",
		Identifier: nil,
	}

	resp, err := handler.Login(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestUserHandler_GetProfile_Success tests successful profile retrieval
func TestUserHandler_GetProfile_Success(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()

	// Register a user first
	registeredUser, err := userSvc.Register(ctx, "testuser", "password123", "test@example.com", "+1234567890")
	assert.NoError(t, err)

	req := &userv1.GetProfileRequest{
		Id: registeredUser.ID,
	}

	resp, err := handler.GetProfile(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.GetUser())
	assert.Equal(t, registeredUser.ID, resp.GetUser().GetId())
	assert.Equal(t, "testuser", resp.GetUser().GetUsername())
}

// TestUserHandler_GetProfile_MissingID tests profile retrieval with missing id
func TestUserHandler_GetProfile_MissingID(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()
	req := &userv1.GetProfileRequest{
		Id: "",
	}

	resp, err := handler.GetProfile(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestUserHandler_UpdateProfile_MissingID tests profile update with missing id
func TestUserHandler_UpdateProfile_MissingID(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()
	req := &userv1.UpdateProfileRequest{
		Id:       "",
		Username: "newusername",
	}

	resp, err := handler.UpdateProfile(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestUserHandler_UpdateProfile_PhoneNumber tests updating only phone number
func TestUserHandler_UpdateProfile_PhoneNumber(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()

	// Register a user first
	registeredUser, err := userSvc.Register(ctx, "testuser", "password123", "test@example.com", "+1111111111")
	assert.NoError(t, err)

	req := &userv1.UpdateProfileRequest{
		Id:          registeredUser.ID,
		Username:    "",
		Email:       "",
		PhoneNumber: "+9999999999",
	}

	resp, err := handler.UpdateProfile(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.GetUser())
	assert.Equal(t, "+9999999999", resp.GetUser().GetPhoneNumber())
}

// TestUserHandler_UpdateVIPAccount_Success tests successful VIP status update
func TestUserHandler_UpdateVIPAccount_Success(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()

	// Register a user first
	registeredUser, err := userSvc.Register(ctx, "testuser", "password123", "test@example.com", "+1234567890")
	assert.NoError(t, err)

	req := &userv1.UpdateVIPAccountRequest{
		Id:    registeredUser.ID,
		IsVip: true,
	}

	resp, err := handler.UpdateVIPAccount(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(200), resp.GetStatus())
	assert.Equal(t, "VIP status updated successfully", resp.GetMessage())
	assert.Equal(t, true, resp.GetCurrentVipStatus())
}

// TestUserHandler_UpdateVIPAccount_MissingID tests VIP update with missing id
func TestUserHandler_UpdateVIPAccount_MissingID(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()
	req := &userv1.UpdateVIPAccountRequest{
		Id:    "",
		IsVip: true,
	}

	resp, err := handler.UpdateVIPAccount(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestUserHandler_ListUsers_Success tests successful user listing
func TestUserHandler_ListUsers_Success(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()

	req := &userv1.ListUsersRequest{}

	resp, err := handler.ListUsers(ctx, req)

	assert.NoError(t, err, "expected no error when listing users")
	assert.NotNil(t, resp, "expected response to not be nil")
	// Just verify the response structure is valid - actual user count depends on mock implementation
	assert.NotNil(t, resp.GetUsers(), "expected users slice to not be nil")
}

// TestUserHandler_DeleteUsers_Success tests successful user deletion
func TestUserHandler_DeleteUsers_Success(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()

	// Register a user
	registeredUser, err := userSvc.Register(ctx, "testuser", "password123", "test@example.com", "+1234567890")
	assert.NoError(t, err)

	req := &userv1.DeleteUsersRequest{
		Ids: []string{registeredUser.ID},
	}

	resp, err := handler.DeleteUsers(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(200), resp.GetStatus())
	assert.Equal(t, "Users deleted successfully", resp.GetMessage())
}

// TestUserHandler_DeleteUsers_MissingIDs tests delete with no ids
func TestUserHandler_DeleteUsers_MissingIDs(t *testing.T) {
	repo := NewMockUserRepository()
	userSvc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())
	handler := handlers.NewUserHandler(userSvc, logger.DefaultNewLogger())

	ctx := context.Background()
	req := &userv1.DeleteUsersRequest{
		Ids: []string{},
	}

	resp, err := handler.DeleteUsers(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}
