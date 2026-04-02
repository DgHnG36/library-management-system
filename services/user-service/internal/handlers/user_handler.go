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
	userService *applications.UserService
	logger      *logger.Logger
}

func NewUserHandler(userService *applications.UserService, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

func (h *UserHandler) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	if req.GetUsername() == "" || req.GetPassword() == "" || req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "username, password and email are required")
	}

	user, err := h.userService.Register(ctx, req.Username, req.Password, req.Email, req.PhoneNumber)
	if err != nil {
		h.logger.Error("Failed to register user", err, logger.Fields{
			"username": req.Username,
		})
		return nil, err
	}

	return &userv1.RegisterResponse{
		Status:  201,
		Message: "Registered successfully",
		UserId:  user.ID,
	}, nil
}

func (h *UserHandler) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	if req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "identifier and password are required")
	}

	var identifier string
	byEmail := false

	switch id := req.Identifier.(type) {
	case *userv1.LoginRequest_Email:
		identifier = id.Email
		byEmail = true
	case *userv1.LoginRequest_Username:
		identifier = id.Username
	default:
		identifier = ""
	}

	if identifier == "" {
		return nil, status.Error(codes.InvalidArgument, "username or email is required")
	}

	user, tokenPair, err := h.userService.Login(ctx, identifier, req.Password, byEmail)
	if err != nil || user == nil {
		h.logger.Error("Failed to login", err, logger.Fields{
			"identifier": identifier,
		})
		return nil, err
	}

	return &userv1.LoginResponse{
		Status:       200,
		Message:      "Login successfully",
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         toPbUser(user),
	}, nil
}

func (h *UserHandler) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.UserProfileResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "ID user is required")
	}

	user, err := h.userService.GetProfile(ctx, req.Id)
	if err != nil || user == nil {
		h.logger.Error("Failed to get profile", err, logger.Fields{
			"user_id": req.Id,
		})
		return nil, err
	}

	return &userv1.UserProfileResponse{
		User: toPbUser(user),
	}, nil
}

func (h *UserHandler) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UserProfileResponse, error) {
	if req.GetId() == "" || ctx.Value("X-User-ID") == "" {
		return nil, status.Error(codes.InvalidArgument, "ID user is required")
	}

	newUsername := req.GetUsername()
	newEmail := req.GetEmail()
	newPhoneNumber := req.GetPhoneNumber()

	user, err := h.userService.UpdateProfile(ctx, req.Id, newUsername, newEmail, newPhoneNumber)
	if err != nil || user == nil {
		h.logger.Error("Failed to update profile", err, logger.Fields{
			"user_id": req.Id,
		})
		return nil, err
	}

	return &userv1.UserProfileResponse{
		User: toPbUser(user),
	}, nil
}

func (h *UserHandler) UpdateVIPAccount(ctx context.Context, req *userv1.UpdateVIPAccountRequest) (*userv1.UpdateVIPAccountResponse, error) {
	userRole := models.UserRole(ctx.Value("X-User-Role").(string))
	if userRole != models.RoleAdmin {
		return nil, status.Error(codes.PermissionDenied, "only admins can update VIP status")
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "ID user is required")
	}

	newVIPStatus := req.GetIsVip()
	currentVipStatus, err := h.userService.UpdateVIPAccount(ctx, req.Id, newVIPStatus)
	if err != nil {
		h.logger.Error("Failed to update VIP account", err, logger.Fields{
			"user_id": req.Id,
		})
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
		page = req.Pagination.GetPage()
		limit = req.Pagination.GetLimit()
		sortBy = req.Pagination.GetSortBy()
		isDesc = req.Pagination.GetIsDesc()
	}

	callerRole := models.UserRole(ctx.Value("X-User-Role").(string))
	if callerRole != models.RoleAdmin && callerRole != models.RoleManager {
		return nil, status.Error(codes.PermissionDenied, "only admins or managers can list users")
	}

	targetRole := models.UserRole(req.GetRole().String())
	users, total, err := h.userService.ListUsers(ctx, callerRole, page, limit, sortBy, isDesc, targetRole)
	if err != nil || users == nil {
		h.logger.Error("Failed to list users", err, logger.Fields{
			"page":  page,
			"limit": limit,
		})
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
	callerRole := models.UserRole(ctx.Value("X-User-Role").(string))
	if callerRole != models.RoleAdmin {
		return nil, status.Error(codes.PermissionDenied, "only admins can delete users")
	}

	if len(req.GetIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ID users are required")
	}

	if err := h.userService.DeleteUsers(ctx, callerRole, req.Ids); err != nil {
		h.logger.Error("Failed to delete users", err, logger.Fields{
			"user_ids": req.Ids,
		})
		return nil, err
	}

	return &commonv1.BaseResponse{
		Status:  200,
		Message: "Users deleted successfully",
	}, nil
}

func (h *UserHandler) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.LoginResponse, error) {
	if req.GetRefreshToken() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "refresh token is required")
	}
	tokenPair, err := h.userService.RefreshToken(ctx, ctx.Value("X-User-ID").(string), req.GetRefreshToken())
	if err != nil {
		h.logger.Error("Failed to refresh token", err)
		return nil, err
	}
	return &userv1.LoginResponse{
		Status:       200,
		Message:      "Generated  refresh token successfully",
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         nil,
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
