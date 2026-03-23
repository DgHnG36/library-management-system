package repository

import (
	"context"

	"github.com/DgHnG36/lib-management-system/services/book-service/internal/models"
)

type BookRepository interface {
	Create(ctx context.Context, books []*models.Book) error
	FindByID(ctx context.Context, id string) (*models.Book, error)
	FindByTitle(ctx context.Context, title string) (*models.Book, error)
	List(ctx context.Context, page, limit int32, sortBy string, isDesc bool, searchQuery, category string) ([]*models.Book, int32, error)
	Update(ctx context.Context, book *models.Book) error
	UpdateAvailableQuantity(ctx context.Context, bookID string, changeAmount int32) (int32, error)
	Delete(ctx context.Context, ids []string) error
}
