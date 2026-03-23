package main

import (
	"context"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/book-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/handlers"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/book-service/pkg/logger"
	commonv1 "github.com/DgHnG36/lib-management-system/shared/go/v1"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newBookHandlerForTest() (*handlers.BookHandler, *MockBookRepository) {
	repo := NewMockBookRepository()
	bookSvc := applications.NewBookService(repo, logger.DefaultNewLogger())
	handler := handlers.NewBookHandler(bookSvc, logger.DefaultNewLogger())
	return handler, repo
}

func TestBookHandler_GetBook_ByID_Success(t *testing.T) {
	handler, repo := newBookHandlerForTest()
	repo.books["book-1"] = &models.Book{
		ID:                "book-1",
		Title:             "Test Book",
		Author:            "Author",
		ISBN:              "123",
		Category:          "Fiction",
		TotalQuantity:     10,
		AvailableQuantity: 10,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	resp, err := handler.GetBook(context.Background(), &bookv1.GetBookRequest{
		Identifier: &bookv1.GetBookRequest_Id{Id: "book-1"},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "book-1", resp.Book.Id)
}

func TestBookHandler_GetBook_MissingIdentifier(t *testing.T) {
	handler, _ := newBookHandlerForTest()

	resp, err := handler.GetBook(context.Background(), &bookv1.GetBookRequest{})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBookHandler_ListBooks_Success(t *testing.T) {
	handler, repo := newBookHandlerForTest()
	repo.books["book-1"] = &models.Book{ID: "book-1", Title: "Book 1", Category: "Fiction", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo.books["book-2"] = &models.Book{ID: "book-2", Title: "Book 2", Category: "History", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	resp, err := handler.ListBooks(context.Background(), &bookv1.ListBooksRequest{
		Pagination: &commonv1.PaginationRequest{Page: 1, Limit: 10},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(2), resp.TotalCount)
	assert.Len(t, resp.Books, 2)
}

func TestBookHandler_CreateBooks_Success(t *testing.T) {
	handler, _ := newBookHandlerForTest()

	resp, err := handler.CreateBooks(context.Background(), &bookv1.CreateBooksRequest{
		Books: []*bookv1.CreateBookPayload{{
			Title:         "New Book",
			Author:        "Author",
			Isbn:          "978123",
			Category:      "Tech",
			Description:   "Desc",
			TotalQuantity: 5,
		}},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(1), resp.SuccessCount)
	assert.Len(t, resp.Books, 1)
}

func TestBookHandler_CreateBooks_EmptyPayload(t *testing.T) {
	handler, _ := newBookHandlerForTest()

	resp, err := handler.CreateBooks(context.Background(), &bookv1.CreateBooksRequest{Books: []*bookv1.CreateBookPayload{}})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBookHandler_UpdateBook_Success(t *testing.T) {
	handler, repo := newBookHandlerForTest()
	repo.books["book-1"] = &models.Book{
		ID:        "book-1",
		Title:     "Old",
		Author:    "A",
		ISBN:      "123",
		Category:  "Fiction",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp, err := handler.UpdateBook(context.Background(), &bookv1.UpdateBookRequest{
		Id:    "book-1",
		Title: "New",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "New", resp.Book.Title)
}

func TestBookHandler_UpdateBook_MissingID(t *testing.T) {
	handler, _ := newBookHandlerForTest()

	resp, err := handler.UpdateBook(context.Background(), &bookv1.UpdateBookRequest{Id: ""})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBookHandler_DeleteBooks_Success(t *testing.T) {
	handler, repo := newBookHandlerForTest()
	repo.books["book-1"] = &models.Book{ID: "book-1", Title: "Book 1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo.books["book-2"] = &models.Book{ID: "book-2", Title: "Book 2", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	resp, err := handler.DeleteBooks(context.Background(), &bookv1.DeleteBooksRequest{Ids: []string{"book-1", "book-2"}})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(200), resp.Status)
	assert.Equal(t, 0, len(repo.books))
}

func TestBookHandler_DeleteBooks_EmptyIDs(t *testing.T) {
	handler, _ := newBookHandlerForTest()

	resp, err := handler.DeleteBooks(context.Background(), &bookv1.DeleteBooksRequest{Ids: []string{}})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBookHandler_CheckAvailability_Success(t *testing.T) {
	handler, repo := newBookHandlerForTest()
	repo.books["book-1"] = &models.Book{ID: "book-1", Title: "Book", AvailableQuantity: 3, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	resp, err := handler.CheckAvailability(context.Background(), &bookv1.CheckAvailabilityRequest{BookId: "book-1"})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.IsAvailable)
	assert.Equal(t, int32(3), resp.AvailableQuantity)
}

func TestBookHandler_CheckAvailability_MissingBookID(t *testing.T) {
	handler, _ := newBookHandlerForTest()

	resp, err := handler.CheckAvailability(context.Background(), &bookv1.CheckAvailabilityRequest{BookId: ""})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBookHandler_UpdateBookQuantity_Success(t *testing.T) {
	handler, repo := newBookHandlerForTest()
	repo.books["book-1"] = &models.Book{ID: "book-1", Title: "Book", AvailableQuantity: 5, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	resp, err := handler.UpdateBookQuantity(context.Background(), &bookv1.UpdateBookQuantityRequest{BookId: "book-1", ChangeAmount: -2})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(3), resp.NewAvailableQuantity)
}

func TestBookHandler_UpdateBookQuantity_MissingBookID(t *testing.T) {
	handler, _ := newBookHandlerForTest()

	resp, err := handler.UpdateBookQuantity(context.Background(), &bookv1.UpdateBookQuantityRequest{BookId: "", ChangeAmount: -1})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}
