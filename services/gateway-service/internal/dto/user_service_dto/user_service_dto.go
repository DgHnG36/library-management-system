package user_service_dto

import (
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/common_dto"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/user_token_dto"
)

type UserDTO struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number,omitempty"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	LastUpdated string `json:"last_updated"`
}

type RegisterRequestDTO struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

type RegisterResponseDTO struct {
	UserID string `json:"user_id"`
}

type LoginRequestDTO struct {
	Identifier string `json:"identifier" binding:"required"` // Can be username or email
	Password   string `json:"password" binding:"required"`
}

type LoginResponseDTO struct {
	TokenPair *user_token_dto.TokenPairDTO `json:"token_pair"`
	User      *UserDTO                     `json:"user"`
}

type GetProfileRequestDTO struct {
	UserID string // set from auth context
}

type UserProfileResponseDTO struct {
	User *UserDTO `json:"user"`
}

type UpdateProfileRequestDTO struct {
	UserID      string // set from auth context
	Username    string `json:"username,omitempty"`
	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

type UpdateVIPAccountRequestDTO struct {
	UserID string `uri:"id" binding:"required"`
	IsVIP  bool   `json:"is_vip"`
}

type UpdateVIPAccountResponseDTO struct {
	CurrentVIPStatus bool `json:"current_vip_status"`
}

type ListUsersRequestDTO struct {
	Pagination *common_dto.PaginationDTO `form:",inline"`
	RoleFilter string                    `form:"role_filter"`
}

type ListUsersResponseDTO struct {
	Users      []*UserDTO `json:"users"`
	TotalCount int32      `json:"total_count"`
}

type DeleteUsersRequestDTO struct {
	UserIDs []string `json:"user_ids" binding:"required"`
}
