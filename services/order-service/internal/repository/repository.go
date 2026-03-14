package repository

import (
	"context"

	"github.com/DgHnG36/lib-management-system/services/order-service/internal/models"
)

type OrderRepository interface {
	Create(ctx context.Context, order *models.Order) error
	FindByID(ctx context.Context, orderID string) (*models.Order, error)
	FindByUserID(ctx context.Context, userID string, page, limit int32, sortBy string, isDesc bool, filterStatus models.OrderStatus) ([]*models.Order, int32, error)
	FindAll(ctx context.Context, page, limit int32, sortBy string, isDesc bool, filterStatus models.OrderStatus, searchUserID string) ([]*models.Order, int32, error)
	UpdateStatus(ctx context.Context, orderID string, newStatus models.OrderStatus, note string) error
	UpdateReturnInfo(ctx context.Context, orderID string, penaltyAmount int32) error
	Cancel(ctx context.Context, orderID, userID, reason string) error
}
