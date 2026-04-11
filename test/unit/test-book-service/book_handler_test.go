package main

import (
	"context"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/book-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/handlers"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/book-service/pkg/interceptor"
	"github.com/DgHnG36/lib-management-system/services/book-service/pkg/logger"
	commonv1 "github.com/DgHnG36/lib-management-system/shared/go/v1"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* HELPER HANDLER FOR TESTING */

func newBookHandlerForTest() (*handlers.BookHandler, *MockBookRepository) {
	repo := NewMockBookRepository()
	bookSvc := applications.NewBookService(repo, logger.DefaultNewLogger())
	handler := handlers.NewBookHandler(bookSvc, testLog)
	return handler, repo
}

/* ctxWithRole injects X-User-Role into a background context using the typed key. */
func ctxWithRole(role string) context.Context {
	return context.WithValue(context.Background(), interceptor.ContextKeyUserRole, role)
}

/* TestBookHandler_GetBook */
/* Verifies: lookups by ID and by title succeed, missing identifier -> InvalidArgument. */

func TestBookHandler_GetBook(t *testing.T) {
	tests := []struct {
		name        string
		req         *bookv1.GetBookRequest
		setup       func(*MockBookRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, r *bookv1.BookResponse)
	}{
		{
			name: "find by ID — correct ID in response",
			req:  &bookv1.GetBookRequest{Identifier: &bookv1.GetBookRequest_Id{Id: "book-1"}},
			setup: func(r *MockBookRepository) {
				r.books["book-1"] = &models.Book{
					ID: "book-1", Title: "Test Book", Author: "Author", ISBN: "123",
					Category: "Fiction", TotalQuantity: 10, AvailableQuantity: 10,
					CreatedAt: time.Now(), UpdatedAt: time.Now(),
				}
			},
			check: func(t *testing.T, r *bookv1.BookResponse) {
				assert.Equal(t, "book-1", r.GetBook().GetId())
				assert.Equal(t, "Test Book", r.GetBook().GetTitle())
			},
		},
		{
			name: "find by title — correct title in response",
			req:  &bookv1.GetBookRequest{Identifier: &bookv1.GetBookRequest_Title{Title: "My Book"}},
			setup: func(r *MockBookRepository) {
				r.books["book-2"] = &models.Book{
					ID: "book-2", Title: "My Book", Author: "Author", ISBN: "456",
					CreatedAt: time.Now(), UpdatedAt: time.Now(),
				}
			},
			check: func(t *testing.T, r *bookv1.BookResponse) {
				assert.Equal(t, "My Book", r.GetBook().GetTitle())
			},
		},
		{
			name:        "missing identifier — InvalidArgument",
			req:         &bookv1.GetBookRequest{},
			wantErrCode: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repo := newBookHandlerForTest()
			if tt.setup != nil {
				tt.setup(repo)
			}
			resp, err := handler.GetBook(context.Background(), tt.req)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

/* TestBookHandler_ListBooks */
/* Verifies: all books returned when no filter, count equals seeded data. */

func TestBookHandler_ListBooks(t *testing.T) {
	tests := []struct {
		name        string
		req         *bookv1.ListBooksRequest
		setup       func(*MockBookRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, r *bookv1.ListBooksResponse)
	}{
		{
			name: "all books returned — TotalCount and len(Books) both equal 2",
			req: &bookv1.ListBooksRequest{
				Pagination: &commonv1.PaginationRequest{Page: 1, Limit: 10},
			},
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", Title: "Book 1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
				r.books["b2"] = &models.Book{ID: "b2", Title: "Book 2", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			check: func(t *testing.T, r *bookv1.ListBooksResponse) {
				assert.Equal(t, int32(2), r.GetTotalCount())
				assert.Len(t, r.GetBooks(), 2)
			},
		},
		{
			name: "filter by category — only Programming books returned",
			req: &bookv1.ListBooksRequest{
				Pagination: &commonv1.PaginationRequest{Page: 1, Limit: 10},
				Category:   "Programming",
			},
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", Title: "Go", Category: "Programming", CreatedAt: time.Now(), UpdatedAt: time.Now()}
				r.books["b2"] = &models.Book{ID: "b2", Title: "History", Category: "History", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			check: func(t *testing.T, r *bookv1.ListBooksResponse) {
				assert.Equal(t, int32(1), r.GetTotalCount())
				assert.Equal(t, "Programming", r.GetBooks()[0].GetCategory())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repo := newBookHandlerForTest()
			if tt.setup != nil {
				tt.setup(repo)
			}
			resp, err := handler.ListBooks(context.Background(), tt.req)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

/* TestBookHandler_CreateBooks */
/* Verifies: single book created (SuccessCount=1), empty payload -> InvalidArgument,
   no role in context -> PermissionDenied. */

func TestBookHandler_CreateBooks(t *testing.T) {
	payload := []*bookv1.CreateBookPayload{
		{Title: "New Book", Author: "Author", Isbn: "978123", Category: "Tech", Description: "Desc", TotalQuantity: 5},
	}
	tests := []struct {
		name        string
		ctx         context.Context
		req         *bookv1.CreateBooksRequest
		wantErrCode codes.Code
		check       func(t *testing.T, r *bookv1.CreateBooksResponse)
	}{
		{
			name: "ADMIN role — single book created, SuccessCount=1",
			ctx:  ctxWithRole("ADMIN"),
			req:  &bookv1.CreateBooksRequest{Books: payload},
			check: func(t *testing.T, r *bookv1.CreateBooksResponse) {
				assert.Equal(t, int32(1), r.GetSuccessCount())
				assert.Len(t, r.GetBooks(), 1)
			},
		},
		{
			name:        "empty payload — InvalidArgument",
			ctx:         ctxWithRole("ADMIN"),
			req:         &bookv1.CreateBooksRequest{Books: []*bookv1.CreateBookPayload{}},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "no role in context — PermissionDenied",
			ctx:         context.Background(),
			req:         &bookv1.CreateBooksRequest{Books: payload},
			wantErrCode: codes.PermissionDenied,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _ := newBookHandlerForTest()
			resp, err := handler.CreateBooks(tt.ctx, tt.req)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

/* TestBookHandler_UpdateBook */
/* Verifies: title updated in response, missing ID -> InvalidArgument,
   no role -> PermissionDenied. */

func TestBookHandler_UpdateBook(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		req         *bookv1.UpdateBookRequest
		setup       func(*MockBookRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, r *bookv1.BookResponse)
	}{
		{
			name: "ADMIN role — updated title reflected in response",
			ctx:  ctxWithRole("ADMIN"),
			req:  &bookv1.UpdateBookRequest{Id: "book-1", Title: "New Title"},
			setup: func(r *MockBookRepository) {
				r.books["book-1"] = &models.Book{
					ID: "book-1", Title: "Old", Author: "A", ISBN: "123",
					Category: "Fiction", CreatedAt: time.Now(), UpdatedAt: time.Now(),
				}
			},
			check: func(t *testing.T, r *bookv1.BookResponse) {
				assert.Equal(t, "New Title", r.GetBook().GetTitle())
			},
		},
		{
			name:        "missing ID — InvalidArgument",
			ctx:         ctxWithRole("ADMIN"),
			req:         &bookv1.UpdateBookRequest{Id: ""},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "no role in context — PermissionDenied",
			ctx:         context.Background(),
			req:         &bookv1.UpdateBookRequest{Id: "book-1", Title: "New"},
			wantErrCode: codes.PermissionDenied,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repo := newBookHandlerForTest()
			if tt.setup != nil {
				tt.setup(repo)
			}
			resp, err := handler.UpdateBook(tt.ctx, tt.req)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

/* TestBookHandler_DeleteBooks */
/* Verifies: Status 200 + repo empty after delete, empty IDs -> InvalidArgument,
   no role -> PermissionDenied. */

func TestBookHandler_DeleteBooks(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		req         *bookv1.DeleteBooksRequest
		setup       func(*MockBookRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, r *commonv1.BaseResponse, repo *MockBookRepository)
	}{
		{
			name: "ADMIN role — Status 200 in response, repo empty afterwards",
			ctx:  ctxWithRole("ADMIN"),
			req:  &bookv1.DeleteBooksRequest{Ids: []string{"b1", "b2"}},
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", Title: "B1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
				r.books["b2"] = &models.Book{ID: "b2", Title: "B2", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			check: func(t *testing.T, r *commonv1.BaseResponse, repo *MockBookRepository) {
				assert.Equal(t, int32(200), r.GetStatus())
				assert.Empty(t, repo.books)
			},
		},
		{
			name:        "empty IDs — InvalidArgument",
			ctx:         ctxWithRole("ADMIN"),
			req:         &bookv1.DeleteBooksRequest{Ids: []string{}},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "no role in context — PermissionDenied",
			ctx:         context.Background(),
			req:         &bookv1.DeleteBooksRequest{Ids: []string{"b1"}},
			wantErrCode: codes.PermissionDenied,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repo := newBookHandlerForTest()
			if tt.setup != nil {
				tt.setup(repo)
			}
			resp, err := handler.DeleteBooks(tt.ctx, tt.req)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp, repo)
			}
		})
	}
}

/* TestBookHandler_CheckAvailability */
/* Verifies: availability flag and quantity returned correctly, missing book_id
   -> InvalidArgument, no role -> PermissionDenied. */

func TestBookHandler_CheckAvailability(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		req         *bookv1.CheckAvailabilityRequest
		setup       func(*MockBookRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, r *bookv1.CheckAvailabilityResponse)
	}{
		{
			name: "SYSTEM role — isAvailable true, quantity 3",
			ctx:  ctxWithRole("SYSTEM"),
			req:  &bookv1.CheckAvailabilityRequest{BookId: "b1"},
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", Title: "Book", AvailableQuantity: 3, CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			check: func(t *testing.T, r *bookv1.CheckAvailabilityResponse) {
				assert.True(t, r.GetIsAvailable())
				assert.Equal(t, int32(3), r.GetAvailableQuantity())
			},
		},
		{
			name:        "missing book_id — InvalidArgument",
			ctx:         ctxWithRole("ADMIN"),
			req:         &bookv1.CheckAvailabilityRequest{BookId: ""},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "no role in context — PermissionDenied",
			ctx:         context.Background(),
			req:         &bookv1.CheckAvailabilityRequest{BookId: "b1"},
			wantErrCode: codes.PermissionDenied,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repo := newBookHandlerForTest()
			if tt.setup != nil {
				tt.setup(repo)
			}
			resp, err := handler.CheckAvailability(tt.ctx, tt.req)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

/* TestBookHandler_UpdateBookQuantity */
/* Verifies: success=true and new quantity after change, missing book_id -> InvalidArgument,
   no role -> PermissionDenied. */

func TestBookHandler_UpdateBookQuantity(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		req         *bookv1.UpdateBookQuantityRequest
		setup       func(*MockBookRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, r *bookv1.UpdateBookQuantityResponse)
	}{
		{
			name: "SYSTEM role — success true, new quantity is 3",
			ctx:  ctxWithRole("SYSTEM"),
			req:  &bookv1.UpdateBookQuantityRequest{BookId: "b1", ChangeAmount: -2},
			setup: func(r *MockBookRepository) {
				r.books["b1"] = &models.Book{ID: "b1", Title: "Book", AvailableQuantity: 5, CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			check: func(t *testing.T, r *bookv1.UpdateBookQuantityResponse) {
				assert.True(t, r.GetSuccess())
				assert.Equal(t, int32(3), r.GetNewAvailableQuantity())
			},
		},
		{
			name:        "missing book_id — InvalidArgument",
			ctx:         ctxWithRole("ADMIN"),
			req:         &bookv1.UpdateBookQuantityRequest{BookId: "", ChangeAmount: -1},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "no role in context — PermissionDenied",
			ctx:         context.Background(),
			req:         &bookv1.UpdateBookQuantityRequest{BookId: "b1", ChangeAmount: -1},
			wantErrCode: codes.PermissionDenied,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repo := newBookHandlerForTest()
			if tt.setup != nil {
				tt.setup(repo)
			}
			resp, err := handler.UpdateBookQuantity(tt.ctx, tt.req)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
