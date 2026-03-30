package repository

import (
	"context"
	"time"

	"github.com/DgHnG36/lib-management-system/services/user-service/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdateVIPStatus(ctx context.Context, id string, isVip bool) error
	List(ctx context.Context, page, limit int32, sortBy string, isDesc bool, role models.UserRole) ([]*models.User, int32, error)
	Delete(ctx context.Context, ids []string) error

	StoreRefreshToken(ctx context.Context, userID, refreshTokenHash string, expiresAt time.Time) error
	FindRefreshToken(ctx context.Context, refreshTokenHashed string) (*models.UserToken, error)
	DeleteRefreshToken(ctx context.Context, refreshTokenHashed string) error
}
