package handlers

import (
	"context"

	"github.com/DgHnG36/lib-management-system/services/user-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/user-service/pkg/logger"
	commonv1 "github.com/DgHnG36/lib-management-system/shared/go/v1"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserHandler struct {
	userv1.UnimplementedUserServiceServer
	userSvc *applications.UserService
	logger  *logger.Logger
}

func NewUserHandler(userSvc *applications.UserService, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		userSvc: userSvc,
		logger:  logger,
	}
}

func (h *UserHandler) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	if req.GetUsername() == "" || req.GetPassword() == "" || req.GetEmail() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "username, password and email are required")
	}

	user, err := h.userSvc.Register(ctx, req.Username, req.Password, req.Email, req.PhoneNumber)
	if err != nil {
		h.logger.Error("Failed to register user", err)
		return nil, err
	}

	return &userv1.RegisterResponse{
		Status:  200,
		Message: "User registered successfully",
		UserId:  user.ID,
	}, nil
}

func (h *UserHandler) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	if req.GetPassword() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password is required")
	}

	var identifier string
	var byEmail bool

	switch id := req.Identifier.(type) {
	case *userv1.LoginRequest_Email:
		identifier = id.Email
		byEmail = true
	case *userv1.LoginRequest_Username:
		identifier = id.Username
		byEmail = false
	default:
		return nil, status.Errorf(codes.InvalidArgument, "username or email is required")
	}

	if identifier == "" {
		return nil, status.Errorf(codes.InvalidArgument, "username or email is required")
	}

	user, token, err := h.userSvc.Login(ctx, identifier, req.GetPassword(), byEmail)
	if err != nil {
		h.logger.Error("Failed to login", err)
		return nil, err
	}

	return &userv1.LoginResponse{
		Status:  200,
		Message: "Login successful",
		Token:   token,
		User:    toPbUser(user),
	}, nil
}

func (h *UserHandler) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.UserProfileResponse, error) {
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id is required")
	}

	user, err := h.userSvc.GetProfile(ctx, req.GetId())
	if err != nil {
		h.logger.Error("Failed to get profile", err)
		return nil, err
	}

	return &userv1.UserProfileResponse{User: toPbUser(user)}, nil
}

func (h *UserHandler) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UserProfileResponse, error) {
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id is required")
	}

	user, err := h.userSvc.UpdateProfile(ctx, req.GetId(), req.GetUsername(), req.GetEmail(), req.GetPhoneNumber())
	if err != nil {
		h.logger.Error("Failed to update profile", err)
		return nil, err
	}

	return &userv1.UserProfileResponse{User: toPbUser(user)}, nil
}

func (h *UserHandler) UpdateVIPAccount(ctx context.Context, req *userv1.UpdateVIPAccountRequest) (*userv1.UpdateVIPAccountResponse, error) {
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id is required")
	}

	currentVipStatus, err := h.userSvc.UpdateVIPAccount(ctx, req.GetId(), req.GetIsVip())
	if err != nil {
		h.logger.Error("Failed to update VIP account", err)
		return nil, err
	}

	return &userv1.UpdateVIPAccountResponse{
		Status:           200,
		Message:          "VIP status updated successfully",
		CurrentVipStatus: currentVipStatus,
	}, nil
}

func (h *UserHandler) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	var page, limit int32 = 1, 10
	var sortBy string
	var isDesc bool

	if req.GetPagination() != nil {
		page = req.GetPagination().GetPage()
		limit = req.GetPagination().GetLimit()
		sortBy = req.GetPagination().GetSortBy()
		isDesc = req.GetPagination().GetIsDesc()
	}

	role := models.UserRole(req.GetRole().String())
	users, total, err := h.userSvc.ListUsers(ctx, page, limit, sortBy, isDesc, role)
	if err != nil {
		h.logger.Error("Failed to list users", err)
		return nil, err
	}

	pbUsers := make([]*userv1.User, len(users))
	for i, u := range users {
		pbUsers[i] = toPbUser(u)
	}

	return &userv1.ListUsersResponse{
		Users:      pbUsers,
		TotalCount: total,
	}, nil
}

func (h *UserHandler) DeleteUsers(ctx context.Context, req *userv1.DeleteUsersRequest) (*commonv1.BaseResponse, error) {
	if len(req.GetIds()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "ids are required")
	}

	if err := h.userSvc.DeleteUsers(ctx, req.GetIds()); err != nil {
		h.logger.Error("Failed to delete users", err)
		return nil, err
	}

	return &commonv1.BaseResponse{
		Status:  200,
		Message: "Users deleted successfully",
	}, nil
}

/* HELPER METHODS */
func toPbUser(user *models.User) *userv1.User {
	return &userv1.User{
		Id:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
		Role:        userv1.UserRole(userv1.UserRole_value[string(user.Role)]),
		IsVip:       user.IsVip,
		IsActive:    user.IsActive,
		CreatedAt:   timestamppb.New(user.CreatedAt),
		LastUpdated: timestamppb.New(user.LastUpdated),
	}
}
