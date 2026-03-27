package repository

import (
	"context"
	"sync"

	"github.com/DgHnG36/lib-management-system/services/order-service/internal/models"
	"gorm.io/gorm"
)

type orderRepo struct {
	db *gorm.DB
}

func NewOrderRepo(db *gorm.DB) OrderRepository {
	return &orderRepo{db: db}
}

func (r *orderRepo) Create(ctx context.Context, order *models.Order) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Books").Create(order).Error; err != nil {
			return err
		}
		if len(order.Books) > 0 {
			if err := tx.Create(&order.Books).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *orderRepo) FindByID(ctx context.Context, orderID string) (*models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).Preload("Books").First(&order, "id = ?", orderID).Error
	if err != nil {
		return nil, nil
	}

	return &order, nil
}

func (r *orderRepo) FindByUserID(ctx context.Context, userID string, page, limit int32, sortBy string, isDesc bool, filterStatus models.OrderStatus) ([]*models.Order, int32, error) {
	var orders []*models.Order
	var total int64
	var countErr, queryErr error

	query := r.db.WithContext(ctx).Model(&models.Order{}).Where("user_id = ?", userID)
	if filterStatus != "" && filterStatus != "STATUS_UNSPECIFIED" {
		query = query.Where("status = ?", filterStatus)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		countErr = query.Session(&gorm.Session{}).Count(&total).Error
	}()

	go func() {
		defer wg.Done()
		offset := (page - 1) * limit
		orderClause := r.buildSortClause(sortBy, isDesc)
		queryErr = query.Session(&gorm.Session{}).
			Preload("Books").
			Order(orderClause).
			Limit(int(limit)).
			Offset(int(offset)).
			Find(&orders).Error
	}()

	wg.Wait()
	if countErr != nil {
		return nil, 0, countErr
	}
	if queryErr != nil {
		return nil, 0, queryErr
	}

	return orders, int32(total), nil
}

func (r *orderRepo) FindAll(ctx context.Context, page, limit int32, sortBy string, isDesc bool, filterStatus models.OrderStatus, searchUserID string) ([]*models.Order, int32, error) {
	var orders []*models.Order
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Order{})
	if filterStatus != "" && filterStatus != "STATUS_UNSPECIFIED" {
		query = query.Where("status = ?", filterStatus)
	}

	if searchUserID != "" {
		query = query.Where("user_id = ?", searchUserID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	orderClause := r.buildSortClause(sortBy, isDesc)
	err := query.Preload("Books").Order(orderClause).Limit(int(limit)).Offset(int(offset)).Find(&orders).Error
	if err != nil {
		return nil, 0, err
	}

	return orders, int32(total), nil
}

func (r *orderRepo) UpdateStatus(ctx context.Context, orderID string, newStatus models.OrderStatus, note string) error {
	result := r.db.WithContext(ctx).Model(&models.Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
		"status": newStatus,
		"note":   note,
	})

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return result.Error
}

func (r *orderRepo) UpdateReturnInfo(ctx context.Context, orderID string, penaltyAmount int32) error {
	result := r.db.WithContext(ctx).Model(&models.Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
		"status":         models.StatusReturned,
		"return_date":    gorm.Expr("NOW()"),
		"penalty_amount": penaltyAmount,
	})

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return result.Error
}

func (r *orderRepo) Cancel(ctx context.Context, orderID, userID, reason string) error {

	result := r.db.WithContext(ctx).Model(&models.Order{}).
		Where("id = ? AND user_id = ? AND status = ?", orderID, userID, models.StatusPending).
		Updates(map[string]interface{}{
			"status":        models.StatusCanceled,
			"cancel_reason": reason,
		})

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return result.Error
}

/* HELPER METHODS */
func (r *orderRepo) buildSortClause(sortBy string, isDesc bool) string {
	if sortBy == "" {
		sortBy = "created_at"
	}
	direction := "ASC"
	if isDesc {
		direction = "DESC"
	}
	return sortBy + " " + direction
}
