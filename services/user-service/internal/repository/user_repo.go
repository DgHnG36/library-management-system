package repository

import (
	"context"

	"github.com/DgHnG36/lib-management-system/services/user-service/internal/models"
	"gorm.io/gorm"
)

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, "username = ?", username).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepo) UpdateVIPStatus(ctx context.Context, id string, isVip bool) error {
	result := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", id).
		Update("is_vip", isVip)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *userRepo) List(ctx context.Context, page, limit int32, sortBy string, isDesc bool, role models.UserRole) ([]*models.User, int32, error) {
	var users []*models.User
	var total int64

	query := r.db.WithContext(ctx).Model(&models.User{})
	if role != "" && role != models.RoleGuest {
		query = query.Where("role = ?", role)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	orderClause := r.buildSortClause(sortBy, isDesc)
	err := query.Order(orderClause).Limit(int(limit)).Offset(int(offset)).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, int32(total), nil
}

func (r *userRepo) Delete(ctx context.Context, ids []string) error {
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.User{}).Error
}

/* HELPER METHODS */
func (r *userRepo) buildSortClause(sortBy string, isDesc bool) string {
	if sortBy == "" {
		sortBy = "created_at"
	}
	direction := "ASC"
	if isDesc {
		direction = "DESC"
	}
	return sortBy + " " + direction
}
