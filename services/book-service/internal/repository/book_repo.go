package repository

import (
	"context"
	"fmt"

	"github.com/DgHnG36/lib-management-system/services/book-service/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type bookRepo struct {
	db *gorm.DB
}

func NewBookRepo(db *gorm.DB) BookRepository {
	return &bookRepo{db: db}
}

func (r *bookRepo) Create(ctx context.Context, books []*models.Book) error {
	return r.db.WithContext(ctx).Create(&books).Error
}

func (r *bookRepo) FindByID(ctx context.Context, id string) (*models.Book, error) {
	var book models.Book
	err := r.db.WithContext(ctx).First(&book, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *bookRepo) FindByTitle(ctx context.Context, title string) (*models.Book, error) {
	var book models.Book
	err := r.db.WithContext(ctx).Where("title ILIKE ?", title).First(&book).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *bookRepo) List(ctx context.Context, page, limit int32, sortBy string, isDesc bool, searchQuery, category string) ([]*models.Book, int32, error) {
	var books []*models.Book
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Book{})

	if searchQuery != "" {
		like := fmt.Sprintf("%%%s%%", searchQuery)
		query = query.Where("title ILIKE ? OR author ILIKE ? OR isbn ILIKE ?", like, like, like)
	}

	if category != "" {
		query = query.Where("category ILIKE ?", category)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	orderClause := r.buildSortClause(sortBy, isDesc)
	err := query.Order(orderClause).Limit(int(limit)).Offset(int(offset)).Find(&books).Error
	if err != nil {
		return nil, 0, err
	}

	return books, int32(total), nil
}

func (r *bookRepo) Update(ctx context.Context, book *models.Book) error {
	return r.db.WithContext(ctx).Save(book).Error
}

func (r *bookRepo) UpdateAvailableQuantity(ctx context.Context, bookID string, changeAmount int32) (int32, error) {
	var book models.Book

	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&book, "id = ?", bookID).Error
	if err == gorm.ErrRecordNotFound {
		return 0, gorm.ErrRecordNotFound
	}
	if err != nil {
		return 0, err
	}

	newQty := book.AvailableQuantity + changeAmount
	if newQty < 0 {
		return 0, fmt.Errorf("insufficient available quantity: current=%d, change=%d", book.AvailableQuantity, changeAmount)
	}
	if newQty > book.TotalQuantity {
		newQty = book.TotalQuantity
	}

	result := r.db.WithContext(ctx).Model(&models.Book{}).
		Where("id = ?", bookID).
		Update("available_quantity", newQty)

	if result.Error != nil {
		return 0, result.Error
	}

	return newQty, nil
}

func (r *bookRepo) Delete(ctx context.Context, ids []string) error {
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.Book{}).Error
}

/* HELPER METHODS */
func (r *bookRepo) buildSortClause(sortBy string, isDesc bool) string {
	if sortBy == "" {
		sortBy = "created_at"
	}
	direction := "ASC"
	if isDesc {
		direction = "DESC"
	}
	return sortBy + " " + direction
}
