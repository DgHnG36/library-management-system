package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/book-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/book-service/pkg/logger"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* ─── Mock Repository ─────────────────────────────────────────────────────── */

/* FailingBookRepository returns a fixed error for every method. */
type FailingBookRepository struct {
	err error
}

func (f *FailingBookRepository) FindByID(_ context.Context, _ string) (*models.Book, error) {
	return nil, f.err
}

func (f *FailingBookRepository) FindByTitle(_ context.Context, _ string) (*models.Book, error) {
	return nil, f.err
}

func (f *FailingBookRepository) List(_ context.Context, _, _ int32, _ string, _ bool, _, _ string) ([]*models.Book, int32, error) {
	return nil, 0, f.err
}

func (f *FailingBookRepository) Create(_ context.Context, _ []*models.Book) error {
	return f.err
}

func (f *FailingBookRepository) Update(_ context.Context, _ *models.Book) error {
	return f.err
}

func (f *FailingBookRepository) UpdateAvailableQuantity(_ context.Context, _ string, _ int32) (int32, error) {
	return 0, f.err
}

func (f *FailingBookRepository) Delete(_ context.Context, _ []string) error {
	return f.err
}

/*
MockBookRepository is an in-memory book store for unit tests.

	Shared across book_service_test.go and book_handler_test.go (same package).
*/
type MockBookRepository struct {
	books map[string]*models.Book
}

func NewMockBookRepository() *MockBookRepository {
	return &MockBookRepository{books: make(map[string]*models.Book)}
}

func (m *MockBookRepository) FindByID(_ context.Context, id string) (*models.Book, error) {
	if book, exists := m.books[id]; exists {
		return book, nil
	}
	return nil, status.Errorf(codes.NotFound, "book not found")
}

func (m *MockBookRepository) FindByTitle(_ context.Context, title string) (*models.Book, error) {
	for _, book := range m.books {
		if book.Title == title {
			return book, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "book not found")
}

func (m *MockBookRepository) List(_ context.Context, _, _ int32, _ string, _ bool, searchQuery, category string) ([]*models.Book, int32, error) {
	var results []*models.Book
	for _, book := range m.books {
		if (searchQuery == "" || book.Title == searchQuery) && (category == "" || book.Category == category) {
			results = append(results, book)
		}
	}
	return results, int32(len(results)), nil
}

func (m *MockBookRepository) Create(_ context.Context, books []*models.Book) error {
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

func (m *MockBookRepository) Update(_ context.Context, book *models.Book) error {
	if _, exists := m.books[book.ID]; !exists {
		return status.Error(codes.NotFound, "book not found")
	}
	book.UpdatedAt = time.Now()
	m.books[book.ID] = book
	return nil
}

func (m *MockBookRepository) UpdateAvailableQuantity(_ context.Context, bookID string, changeAmount int32) (int32, error) {
	book, exists := m.books[bookID]
	if !exists {
		return 0, status.Error(codes.NotFound, "book not found")
	}
	book.AvailableQuantity += changeAmount
	return book.AvailableQuantity, nil
}

func (m *MockBookRepository) Delete(_ context.Context, ids []string) error {
	for _, id := range ids {
		if _, exists := m.books[id]; !exists {
			return status.Error(codes.NotFound, "book not found")
		}
		delete(m.books, id)
	}
	return nil
}

/* ─── Shared test helpers ─────────────────────────────────────────────────── */

var testLog = logger.DefaultNewLogger()

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func newTestSvc() (*applications.BookService, *MockBookRepository) {
	repo := NewMockBookRepository()
	svc := applications.NewBookService(repo, testLog)
	return svc, repo
}

/* ─── TestBookService_GetBook ─────────────────────────────────────────────── */
/* Verifies: lookup by ID, lookup by title, not-found → Internal, both args
   empty → InvalidArgument. */

func TestBookService_GetBook(t *testing.T) {
	seed := &models.Book{
		ID: "book-1", Title: "Unique Title", Author: "Author",
		ISBN: "123456", Category: "Fiction", TotalQuantity: 10,
		AvailableQuantity: 10, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	tests := []struct {
		name        string
		id          string
		title       string
		wantErrCode codes.Code
		check       func(t *testing.T, b *models.Book)
	}{
		{
			name: "find by ID — correct book returned",
			id:   "book-1",
			check: func(t *testing.T, b *models.Book) {
				assert.Equal(t, "book-1", b.ID)
				assert.Equal(t, "Unique Title", b.Title)
			},
		},
		{
			name:  "find by title — correct book returned",
			title: "Unique Title",
			check: func(t *testing.T, b *models.Book) {
				assert.Equal(t, "Unique Title", b.Title)
			},
		},
		{
			name:        "not found by ID — Internal",
			id:          "nonexistent",
			wantErrCode: codes.Internal,
		},
		{
			name:        "both ID and title empty — InvalidArgument",
			wantErrCode: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			repo.books["book-1"] = seed

			b, err := svc.GetBook(context.Background(), tt.id, tt.title)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, b)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, b)
			if tt.check != nil {
				tt.check(t, b)
			}
		})
	}
}

/* ─── TestBookService_ListBooks ───────────────────────────────────────────── */
/* Verifies: all books returned, filter by searchQuery, filter by category,
   invalid page/limit values default to 1/10. */

func TestBookService_ListBooks(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*MockBookRepository)
		page, limit int32
		searchQuery string
		category    string
		wantErrCode codes.Code
		check       func(t *testing.T, books []*models.Book, total int32)
	}{
		{
			name: "all books — total and len match",
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", Title: "Book 1", Category: "Fiction"}
				r.books["b2"] = &models.Book{ID: "b2", Title: "Book 2", Category: "History"}
			},
			page: 1, limit: 10,
			check: func(t *testing.T, books []*models.Book, total int32) {
				assert.Equal(t, int32(2), total)
				assert.Len(t, books, 2)
			},
		},
		{
			name: "filter by searchQuery — only matching title returned",
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", Title: "Golang Guide"}
				r.books["b2"] = &models.Book{ID: "b2", Title: "Python Guide"}
			},
			page: 1, limit: 10, searchQuery: "Golang Guide",
			check: func(t *testing.T, books []*models.Book, total int32) {
				assert.Equal(t, int32(1), total)
				assert.Len(t, books, 1)
				assert.Equal(t, "Golang Guide", books[0].Title)
			},
		},
		{
			name: "filter by category — only matching category returned",
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", Title: "Golang Book", Category: "Programming"}
				r.books["b2"] = &models.Book{ID: "b2", Title: "History Book", Category: "History"}
			},
			page: 1, limit: 10, category: "Programming",
			check: func(t *testing.T, books []*models.Book, total int32) {
				assert.Equal(t, int32(1), total)
				assert.Equal(t, "Programming", books[0].Category)
			},
		},
		{
			name: "invalid page and limit — default to 1/10, result still returned",
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", Title: "Book 1"}
			},
			page: -1, limit: 0,
			check: func(t *testing.T, books []*models.Book, total int32) {
				assert.Equal(t, int32(1), total)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			books, total, err := svc.ListBooks(context.Background(), tt.page, tt.limit, "title", false, tt.searchQuery, tt.category)
			if tt.wantErrCode != codes.OK {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, books, total)
			}
		})
	}
}

/* ─── TestBookService_CreateBooks ─────────────────────────────────────────── */
/* Verifies: single creation with generated ID, multiple books with distinct
   IDs, empty title rejected with Internal (repo layer). */

func TestBookService_CreateBooks(t *testing.T) {
	tests := []struct {
		name        string
		payloads    []applications.CreateBookPayload
		wantErrCode codes.Code
		check       func(t *testing.T, books []*models.Book)
	}{
		{
			name: "single book — ID generated, AvailableQuantity equals TotalQuantity, CreatedAt set",
			payloads: []applications.CreateBookPayload{
				{Title: "New Book", Author: "Author", ISBN: "9876543", Category: "Fiction", TotalQuantity: 5, Description: "Desc"},
			},
			check: func(t *testing.T, books []*models.Book) {
				assert.Len(t, books, 1)
				assert.NotEmpty(t, books[0].ID)
				assert.Equal(t, "New Book", books[0].Title)
				assert.Equal(t, int32(5), books[0].AvailableQuantity)
				assert.False(t, books[0].CreatedAt.IsZero())
			},
		},
		{
			name: "multiple books — all created with distinct IDs",
			payloads: []applications.CreateBookPayload{
				{Title: "Book 1", Author: "A1", ISBN: "111", Category: "Fiction", TotalQuantity: 10},
				{Title: "Book 2", Author: "A2", ISBN: "222", Category: "Non-Fiction", TotalQuantity: 5},
			},
			check: func(t *testing.T, books []*models.Book) {
				assert.Len(t, books, 2)
				assert.NotEqual(t, books[0].ID, books[1].ID)
			},
		},
		{
			name: "empty title — Internal (repo rejects empty title)",
			payloads: []applications.CreateBookPayload{
				{Title: "", Author: "A", ISBN: "123", TotalQuantity: 5},
			},
			wantErrCode: codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestSvc()
			books, err := svc.CreateBooks(context.Background(), tt.payloads)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, books)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, books)
			}
		})
	}
}

/* ─── TestBookService_UpdateBook ──────────────────────────────────────────── */
/* Verifies: all fields updated, blank fields keep original values, not-found
   → Internal. */

func TestBookService_UpdateBook(t *testing.T) {
	tests := []struct {
		name                                       string
		bookID                                     string
		title, author, isbn, category, description string
		setup                                      func(*MockBookRepository)
		wantErrCode                                codes.Code
		check                                      func(t *testing.T, b *models.Book)
	}{
		{
			name:   "full update — all new values reflected in returned book",
			bookID: "book-1",
			title:  "New Title", author: "New Author", isbn: "9876543", category: "Non-Fiction", description: "New Desc",
			setup: func(r *MockBookRepository) {
				r.books["book-1"] = &models.Book{
					ID: "book-1", Title: "Old", Author: "Old A", ISBN: "123456",
					Category: "Fiction", CreatedAt: time.Now(), UpdatedAt: time.Now(),
				}
			},
			check: func(t *testing.T, b *models.Book) {
				assert.Equal(t, "New Title", b.Title)
				assert.Equal(t, "New Author", b.Author)
				assert.Equal(t, "9876543", b.ISBN)
				assert.Equal(t, "Non-Fiction", b.Category)
				assert.Equal(t, "New Desc", b.Description)
			},
		},
		{
			name:   "partial update — blank args preserve original field values",
			bookID: "book-1",
			title:  "New Title",
			setup: func(r *MockBookRepository) {
				r.books["book-1"] = &models.Book{
					ID: "book-1", Title: "Original", Author: "Original Author",
					ISBN: "123456", Category: "Fiction", CreatedAt: time.Now(), UpdatedAt: time.Now(),
				}
			},
			check: func(t *testing.T, b *models.Book) {
				assert.Equal(t, "New Title", b.Title)
				assert.Equal(t, "Original Author", b.Author)
			},
		},
		{
			name:        "not found — Internal",
			bookID:      "nonexistent",
			title:       "Title",
			wantErrCode: codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			b, err := svc.UpdateBook(context.Background(), tt.bookID, tt.title, tt.author, tt.isbn, tt.category, tt.description)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, b)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, b)
			if tt.check != nil {
				tt.check(t, b)
			}
		})
	}
}

/* ─── TestBookService_DeleteBooks ─────────────────────────────────────────── */
/* Verifies: successful deletion empties repo, non-existent ID → Internal. */

func TestBookService_DeleteBooks(t *testing.T) {
	tests := []struct {
		name        string
		ids         []string
		setup       func(*MockBookRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, repo *MockBookRepository)
	}{
		{
			name: "delete existing books — repo empty afterwards",
			ids:  []string{"b1", "b2"},
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", Title: "Book 1"}
				r.books["b2"] = &models.Book{ID: "b2", Title: "Book 2"}
			},
			check: func(t *testing.T, repo *MockBookRepository) {
				assert.Empty(t, repo.books)
			},
		},
		{
			name:        "non-existing ID — Internal",
			ids:         []string{"nonexistent"},
			wantErrCode: codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			err := svc.DeleteBooks(context.Background(), tt.ids)
			if tt.wantErrCode != codes.OK {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, repo)
			}
		})
	}
}

/* ─── TestBookService_CheckAvailability ───────────────────────────────────── */
/* Verifies: available book returns true + correct qty, zero qty returns false,
   not-found → Internal. */

func TestBookService_CheckAvailability(t *testing.T) {
	tests := []struct {
		name        string
		bookID      string
		setup       func(*MockBookRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, available bool, qty int32)
	}{
		{
			name:   "available — isAvailable true, quantity 5",
			bookID: "b1",
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", Title: "Book", AvailableQuantity: 5}
			},
			check: func(t *testing.T, available bool, qty int32) {
				assert.True(t, available)
				assert.Equal(t, int32(5), qty)
			},
		},
		{
			name:   "zero quantity — isAvailable false",
			bookID: "b1",
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", Title: "Book", AvailableQuantity: 0}
			},
			check: func(t *testing.T, available bool, qty int32) {
				assert.False(t, available)
				assert.Equal(t, int32(0), qty)
			},
		},
		{
			name:        "not found — Internal",
			bookID:      "nonexistent",
			wantErrCode: codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			available, qty, err := svc.CheckAvailability(context.Background(), tt.bookID)
			if tt.wantErrCode != codes.OK {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, available, qty)
			}
		})
	}
}

/* ─── TestBookService_UpdateBookQuantity ──────────────────────────────────── */
/* Verifies: correct new quantity after decrement/increment, non-existent book
   → Internal. */

func TestBookService_UpdateBookQuantity(t *testing.T) {
	tests := []struct {
		name         string
		bookID       string
		changeAmount int32
		setup        func(*MockBookRepository)
		wantErrCode  codes.Code
		check        func(t *testing.T, newQty int32)
	}{
		{
			name:         "decrement by 1 — new quantity is 3",
			bookID:       "b1",
			changeAmount: -1,
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", AvailableQuantity: 4}
			},
			check: func(t *testing.T, newQty int32) {
				assert.Equal(t, int32(3), newQty)
			},
		},
		{
			name:         "increment by 2 — new quantity is 7",
			bookID:       "b1",
			changeAmount: 2,
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", AvailableQuantity: 5}
			},
			check: func(t *testing.T, newQty int32) {
				assert.Equal(t, int32(7), newQty)
			},
		},
		{
			name:         "non-existing book — Internal",
			bookID:       "nonexistent",
			changeAmount: -1,
			wantErrCode:  codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			newQty, err := svc.UpdateBookQuantity(context.Background(), tt.bookID, tt.changeAmount)
			if tt.wantErrCode != codes.OK {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, newQty)
			}
		})
	}
}
