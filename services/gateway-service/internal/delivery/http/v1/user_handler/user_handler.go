package user_handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/clients/user_service_client"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/mapper"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/user_service_dto"
	pkgerrors "github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/errors"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type UserClientInterface interface {
	Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error)
	Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error)
	GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.UserProfileResponse, error)
	UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UserProfileResponse, error)
	UpdateVIPAccount(ctx context.Context, req *userv1.UpdateVIPAccountRequest) (*userv1.UpdateVIPAccountResponse, error)
	ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error)
	DeleteUsers(ctx context.Context, req *userv1.DeleteUsersRequest) error
	GetConnection() *grpc.ClientConn
}

type UserHandler struct {
	userServiceClient UserClientInterface
	mapper            mapper.MapperInterface
	logger            *logger.Logger
}

func NewUserHandler(addr string, log *logger.Logger) *UserHandler {
	userServiceClient, err := user_service_client.NewUserServiceClient(addr, log)
	if err != nil {
		log.Fatal("Failed to create user service client", err, logger.Fields{
			"address": addr,
		})

		return nil
	}

	mapper := mapper.NewMapper()

	return &UserHandler{
		userServiceClient: userServiceClient,
		mapper:            mapper,
		logger:            log,
	}
}

func NewUserHandlerWithClient(client UserClientInterface, m mapper.MapperInterface, log *logger.Logger) *UserHandler {
	return &UserHandler{
		userServiceClient: client,
		mapper:            m,
		logger:            log,
	}
}

func (h *UserHandler) Close() {
	if h.userServiceClient != nil && h.userServiceClient.GetConnection() != nil {
		if err := h.userServiceClient.GetConnection().Close(); err != nil {
			h.logger.Error("Failed to close user service client connection", err)
		}
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var req user_service_dto.RegisterRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind register request", err)
		c.JSON(400, gin.H{
			"error": "Invalid request payload",
		})
		return
	}

	grpcReq := h.mapper.MapPbRegisterRequest(&req)
	resp, err := h.userServiceClient.Register(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to register user", err, logger.Fields{
			"username":     req.Username,
			"email":        req.Email,
			"phone_number": req.PhoneNumber,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTORegisterResponse(resp)
	c.JSON(201, httpResp)
}

func (h *UserHandler) Login(c *gin.Context) {
	var req user_service_dto.LoginRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind login request", err)
		c.JSON(400, gin.H{
			"error": "Invalid request payload",
		})
		return
	}

	isEmail := h.identifyEmail(req.Identifier)

	grpcReq := h.mapper.MapPbLoginRequest(&req, isEmail)
	resp, err := h.userServiceClient.Login(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to login user", err, logger.Fields{
			"identifier": req.Identifier,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOLoginResponse(resp)
	c.JSON(200, httpResp)
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	var req user_service_dto.GetProfileRequestDTO
	req.UserID = c.GetString("X-User-ID")

	grpcReq := h.mapper.MapPbGetProfileRequest(&req)
	resp, err := h.userServiceClient.GetProfile(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to get user profile", err, logger.Fields{
			"user_id": req.UserID,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOGetProfileResponse(resp)
	c.JSON(200, httpResp)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req user_service_dto.UpdateProfileRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind update profile request", err)
		c.JSON(400, gin.H{
			"error": "Invalid request payload",
		})
		return
	}

	req.UserID = c.GetString("X-User-ID")

	grpcReq := h.mapper.MapPbUpdateProfileRequest(&req)
	resp, err := h.userServiceClient.UpdateProfile(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to update user profile", err, logger.Fields{
			"user_id":      req.UserID,
			"username":     req.Username,
			"email":        req.Email,
			"phone_number": req.PhoneNumber,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOUpdateProfileResponse(resp)
	c.JSON(200, httpResp)
}

func (h *UserHandler) UpdateVIPAccount(c *gin.Context) {
	var req user_service_dto.UpdateVIPAccountRequestDTO
	if err := c.ShouldBindUri(&req); err != nil {
		h.logger.Error("Failed to bind update VIP account request (uri)", err)
		c.JSON(400, gin.H{
			"error": "Invalid request parameters",
		})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind update VIP account request (body)", err)
		c.JSON(400, gin.H{
			"error": "Invalid request payload",
		})
		return
	}

	grpcReq := h.mapper.MapPbUpdateVIPAccountRequest(&req)
	resp, err := h.userServiceClient.UpdateVIPAccount(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to update VIP account", err, logger.Fields{
			"user_id": req.UserID,
			"is_vip":  req.IsVIP,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOUpdateVIPAccountResponse(resp)
	c.JSON(200, httpResp)
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	var req user_service_dto.ListUsersRequestDTO
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("Failed to bind list users request", err)
		c.JSON(400, gin.H{
			"error": "Invalid query parameters",
		})
		return
	}

	grpcReq := h.mapper.MapPbListUsersRequest(&req)
	resp, err := h.userServiceClient.ListUsers(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to list users", err, logger.Fields{
			"role_filter": req.RoleFilter,
			"page":        req.Pagination.Page,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOListUsersResponse(resp)
	c.JSON(200, httpResp)
}

func (h *UserHandler) DeleteUsers(c *gin.Context) {
	var req user_service_dto.DeleteUsersRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind delete users request", err)
		c.JSON(400, gin.H{
			"error": "Invalid request payload",
		})
		return
	}

	grpcReq := h.mapper.MapPbDeleteUsersRequest(&req)
	err := h.userServiceClient.DeleteUsers(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to delete users", err, logger.Fields{
			"user_ids": req.UserIDs,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	c.JSON(204, nil)
}

func (h *UserHandler) CheckConnection() (bool, error) {
	if h.userServiceClient == nil || h.userServiceClient.GetConnection() == nil {
		return false, fmt.Errorf("user service client is not initialized")
	}

	if h.userServiceClient.GetConnection().GetState() != connectivity.Ready {
		return false, fmt.Errorf("user service is not ready")
	}

	return true, nil
}

/* HELPER METHOD */
func (h *UserHandler) identifyEmail(identifier string) bool {
	return strings.Contains(identifier, "@") // Simple check to determine if the identifier is an email or username
}
