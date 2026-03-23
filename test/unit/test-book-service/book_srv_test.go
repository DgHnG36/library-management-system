package main

import (
	"context"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/book-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/book-service/pkg/logger"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MockBookRepository struct {
	books map[string]*models.Book
}

func NewMockBookRepository() *MockBookRepository {
	return &MockBookRepository{
		books: make(map[string]*models.Book),
	}
}

func (m *MockBookRepository) FindByID(ctx context.Context, id string) (*models.Book, error) {
	if book, exists := m.books[id]; exists {
		return book, nil
	}
	return nil, status.Errorf(codes.NotFound, "book not found")
}

func (m *MockBookRepository) FindByTitle(ctx context.Context, title string) (*models.Book, error) {
	for _, book := range m.books {
		if book.Title == title {
			return book, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "book not found")
}

func (m *MockBookRepository) List(ctx context.Context, page, limit int32, sortBy string, isDesc bool, searchQuery, category string) ([]*models.Book, int32, error) {
	var results []*models.Book
	for _, book := range m.books {
		if (searchQuery == "" || book.Title == searchQuery) && (category == "" || book.Category == category) {
			results = append(results, book)
		}
	}
	return results, int32(len(results)), nil
}

func (m *MockBookRepository) Create(ctx context.Context, books []*models.Book) error {
	for _, book := range books {
		if book.ID == "" {
			return status.Error(codes.InvalidArgument, "id is required")
		}
		if book.Title == "" {
			return status.Error(codes.InvalidArgument, "title is required")
		}
		if _, exists := m.books[book.ID]; exists {
			return status.Error(codes.AlreadyExists, "book already exists")
		}
		m.books[book.ID] = book
	}
	return nil
}

func (m *MockBookRepository) Update(ctx context.Context, book *models.Book) error {
	if _, exists := m.books[book.ID]; !exists {
		return status.Error(codes.NotFound, "book not found")
	}
	book.UpdatedAt = time.Now()
	m.books[book.ID] = book
	return nil
}

func (m *MockBookRepository) UpdateAvailableQuantity(ctx context.Context, bookID string, changeAmount int32) (int32, error) {
	book, exists := m.books[bookID]
	if !exists {
		return 0, status.Error(codes.NotFound, "book not found")
	}
	book.AvailableQuantity += changeAmount
	return book.AvailableQuantity, nil
}

func (m *MockBookRepository) Delete(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if _, exists := m.books[id]; !exists {
			return status.Error(codes.NotFound, "book not found")
		}
		delete(m.books, id)
	}
	return nil
}

func TestBookService_GetBook_ByID(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	book := &models.Book{
		ID:                "book-1",
		Title:             "Test Book",
		Author:            "Author",
		ISBN:              "123456",
		Category:          "Fiction",
		TotalQuantity:     10,
		AvailableQuantity: 10,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	repo.books["book-1"] = book

	result, err := service.GetBook(ctx, "book-1", "")
	assert.NoError(t, err)
	assert.Equal(t, book.ID, result.ID)
	assert.Equal(t, "Test Book", result.Title)
}

func TestBookService_GetBook_ByTitle(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	book := &models.Book{
		ID:                "book-1",
		Title:             "Unique Title",
		Author:            "Author",
		ISBN:              "123456",
		Category:          "Fiction",
		TotalQuantity:     10,
		AvailableQuantity: 10,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	repo.books["book-1"] = book

	result, err := service.GetBook(ctx, "", "Unique Title")
	assert.NoError(t, err)
	assert.Equal(t, "Unique Title", result.Title)
}

func TestBookService_GetBook_NotFound(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	_, err := service.GetBook(ctx, "nonexistent", "")
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestBookService_GetBook_MissingBothIDAndTitle(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	_, err := service.GetBook(ctx, "", "")
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBookService_ListBooks_Success(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	repo.books["book-1"] = &models.Book{
		ID:       "book-1",
		Title:    "Book 1",
		Category: "Fiction",
	}
	repo.books["book-2"] = &models.Book{
		ID:       "book-2",
		Title:    "Book 2",
		Category: "Non-Fiction",
	}

	results, total, err := service.ListBooks(ctx, 1, 10, "title", false, "", "")
	assert.NoError(t, err)
	assert.Equal(t, int32(2), total)
	assert.Len(t, results, 2)
}

func TestBookService_ListBooks_WithSearch(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	repo.books["book-1"] = &models.Book{
		ID:    "book-1",
		Title: "Golang Guide",
	}
	repo.books["book-2"] = &models.Book{
		ID:    "book-2",
		Title: "Python Guide",
	}

	results, total, err := service.ListBooks(ctx, 1, 10, "title", false, "Golang Guide", "")
	assert.NoError(t, err)
	assert.Equal(t, int32(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, "Golang Guide", results[0].Title)
}

func TestBookService_ListBooks_WithCategory(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	repo.books["book-1"] = &models.Book{
		ID:       "book-1",
		Title:    "Golang Book",
		Category: "Programming",
	}
	repo.books["book-2"] = &models.Book{
		ID:       "book-2",
		Title:    "History Book",
		Category: "History",
	}

	results, total, err := service.ListBooks(ctx, 1, 10, "title", false, "", "Programming")
	assert.NoError(t, err)
	assert.Equal(t, int32(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, "Programming", results[0].Category)
}

func TestBookService_CreateBooks_Success(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	payloads := []applications.CreateBookPayload{
		{
			Title:         "New Book",
			Author:        "New Author",
			ISBN:          "9876543",
			Category:      "Fiction",
			TotalQuantity: 5,
			Description:   "A great book",
		},
	}

	results, err := service.CreateBooks(ctx, payloads)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NotEmpty(t, results[0].ID)
	assert.Equal(t, "New Book", results[0].Title)
	assert.NotZero(t, results[0].CreatedAt)
}

func TestBookService_CreateBooks_MultipleBooks(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	payloads := []applications.CreateBookPayload{
		{
			Title:         "Book 1",
			Author:        "Author 1",
			ISBN:          "111111",
			Category:      "Fiction",
			TotalQuantity: 10,
		},
		{
			Title:         "Book 2",
			Author:        "Author 2",
			ISBN:          "222222",
			Category:      "Non-Fiction",
			TotalQuantity: 5,
		},
	}

	results, err := service.CreateBooks(ctx, payloads)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NotEqual(t, results[0].ID, results[1].ID)
}

func TestBookService_CreateBooks_EmptyTitle(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	payloads := []applications.CreateBookPayload{
		{
			Title:         "",
			Author:        "Author",
			ISBN:          "123456",
			TotalQuantity: 5,
		},
	}

	_, err := service.CreateBooks(ctx, payloads)
	assert.Error(t, err)
}

func TestBookService_UpdateBook_Success(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	book := &models.Book{
		ID:        "book-1",
		Title:     "Old Title",
		Author:    "Old Author",
		ISBN:      "123456",
		Category:  "Fiction",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.books["book-1"] = book

	result, err := service.UpdateBook(ctx, "book-1", "New Title", "New Author", "9876543", "Non-Fiction", "New Description")
	assert.NoError(t, err)
	assert.Equal(t, "New Title", result.Title)
	assert.Equal(t, "New Author", result.Author)
	assert.Equal(t, "9876543", result.ISBN)
	assert.Equal(t, "Non-Fiction", result.Category)
	assert.Equal(t, "New Description", result.Description)
}

func TestBookService_UpdateBook_PartialUpdate(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	originalAuthor := "Original Author"
	book := &models.Book{
		ID:        "book-1",
		Title:     "Original Title",
		Author:    originalAuthor,
		ISBN:      "123456",
		Category:  "Fiction",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.books["book-1"] = book

	result, err := service.UpdateBook(ctx, "book-1", "New Title", "", "", "", "")
	assert.NoError(t, err)
	assert.Equal(t, "New Title", result.Title)
	assert.Equal(t, originalAuthor, result.Author)
}

func TestBookService_UpdateBook_NotFound(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	_, err := service.UpdateBook(ctx, "nonexistent", "New Title", "", "", "", "")
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestBookService_DeleteBooks_Success(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	repo.books["book-1"] = &models.Book{
		ID:    "book-1",
		Title: "Book",
	}
	repo.books["book-2"] = &models.Book{
		ID:    "book-2",
		Title: "Book 2",
	}

	err := service.DeleteBooks(ctx, []string{"book-1", "book-2"})
	assert.NoError(t, err)
	assert.Len(t, repo.books, 0)
}

func TestBookService_DeleteBooks_NotFound(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	err := service.DeleteBooks(ctx, []string{"nonexistent"})
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestBookService_CheckAvailability_Available(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	book := &models.Book{
		ID:                "book-1",
		Title:             "Book",
		AvailableQuantity: 5,
	}
	repo.books["book-1"] = book

	available, quantity, err := service.CheckAvailability(ctx, "book-1")
	assert.NoError(t, err)
	assert.True(t, available)
	assert.Equal(t, int32(5), quantity)
}

func TestBookService_CheckAvailability_NotAvailable(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	book := &models.Book{
		ID:                "book-1",
		Title:             "Book",
		AvailableQuantity: 0,
	}
	repo.books["book-1"] = book

	available, quantity, err := service.CheckAvailability(ctx, "book-1")
	assert.NoError(t, err)
	assert.False(t, available)
	assert.Equal(t, int32(0), quantity)
}

func TestBookService_CheckAvailability_NotFound(t *testing.T) {
	repo := NewMockBookRepository()
	log := logger.DefaultNewLogger()
	service := applications.NewBookService(repo, log)
	ctx := context.Background()

	_, _, err := service.CheckAvailability(ctx, "nonexistent")
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}
