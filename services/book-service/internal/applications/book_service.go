package applications

import (
	"context"
	"time"

	"github.com/DgHnG36/lib-management-system/services/book-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/repository"
	"github.com/DgHnG36/lib-management-system/services/book-service/pkg/logger"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type BookService struct {
	bookRepo repository.BookRepository
	logger   *logger.Logger
}

func NewBookService(bookRepo repository.BookRepository, logger *logger.Logger) *BookService {
	return &BookService{
		bookRepo: bookRepo,
		logger:   logger,
	}
}

func (s *BookService) GetBook(ctx context.Context, id, title string) (*models.Book, error) {
	var (
		book *models.Book
		err  error
	)

	if id != "" {
		book, err = s.bookRepo.FindByID(ctx, id)
	} else if title != "" {
		book, err = s.bookRepo.FindByTitle(ctx, title)
	} else {
		return nil, status.Errorf(codes.InvalidArgument, "id or title is required")
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get book: %v", err)
	}
	if book == nil {
		return nil, status.Errorf(codes.NotFound, "book not found")
	}

	return book, nil
}

func (s *BookService) ListBooks(ctx context.Context, page, limit int32, sortBy string, isDesc bool, searchQuery, category string) ([]*models.Book, int32, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	books, total, err := s.bookRepo.List(ctx, page, limit, sortBy, isDesc, searchQuery, category)
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, "failed to list books: %v", err)
	}

	return books, total, nil
}

func (s *BookService) CreateBooks(ctx context.Context, payloads []CreateBookPayload) ([]*models.Book, error) {
	s.logger.Info("Creating books", logger.Fields{"count": len(payloads)})

	now := time.Now().UTC()
	books := make([]*models.Book, len(payloads))
	for i, p := range payloads {
		books[i] = &models.Book{
			ID:                uuid.New().String(),
			Title:             p.Title,
			Author:            p.Author,
			ISBN:              p.ISBN,
			Category:          p.Category,
			Description:       p.Description,
			TotalQuantity:     p.TotalQuantity,
			AvailableQuantity: p.TotalQuantity, // initially all available
			CreatedAt:         now,
			UpdatedAt:         now,
		}
	}

	if err := s.bookRepo.Create(ctx, books); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create books: %v", err)
	}

	s.logger.Info("Books created", logger.Fields{"count": len(books)})
	return books, nil
}

func (s *BookService) UpdateBook(ctx context.Context, id, title, author, isbn, category, description string) (*models.Book, error) {
	s.logger.Info("Updating book", logger.Fields{"book_id": id})

	book, err := s.bookRepo.FindByID(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find book: %v", err)
	}
	if book == nil {
		return nil, status.Errorf(codes.NotFound, "book %s not found", id)
	}

	if title != "" {
		book.Title = title
	}
	if author != "" {
		book.Author = author
	}
	if isbn != "" {
		book.ISBN = isbn
	}
	if category != "" {
		book.Category = category
	}
	if description != "" {
		book.Description = description
	}

	if err := s.bookRepo.Update(ctx, book); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update book: %v", err)
	}

	return book, nil
}

func (s *BookService) DeleteBooks(ctx context.Context, ids []string) error {
	s.logger.Info("Deleting books", logger.Fields{"ids": ids})

	if err := s.bookRepo.Delete(ctx, ids); err != nil {
		return status.Errorf(codes.Internal, "failed to delete books: %v", err)
	}
	return nil
}

func (s *BookService) CheckAvailability(ctx context.Context, bookID string) (bool, int32, error) {
	book, err := s.bookRepo.FindByID(ctx, bookID)
	if err != nil {
		return false, 0, status.Errorf(codes.Internal, "failed to find book: %v", err)
	}
	if book == nil {
		return false, 0, status.Errorf(codes.NotFound, "book %s not found", bookID)
	}

	return book.AvailableQuantity > 0, book.AvailableQuantity, nil
}

func (s *BookService) UpdateBookQuantity(ctx context.Context, bookID string, changeAmount int32) (int32, error) {
	s.logger.Info("Updating book quantity", logger.Fields{
		"book_id":       bookID,
		"change_amount": changeAmount,
	})

	newQty, err := s.bookRepo.UpdateAvailableQuantity(ctx, bookID, changeAmount)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, status.Errorf(codes.NotFound, "book %s not found", bookID)
		}
		return 0, status.Errorf(codes.Internal, "failed to update book quantity: %v", err)
	}

	return newQty, nil
}

/* PAYLOAD TYPE */
type CreateBookPayload struct {
	Title         string
	Author        string
	ISBN          string
	Category      string
	Description   string
	TotalQuantity int32
}
