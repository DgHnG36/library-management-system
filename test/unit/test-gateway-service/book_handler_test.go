package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/delivery/http/v1/book_handler"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* MOCK BOOK SERVICE CLIENT — implements book_handler.BookClientInterface */

type Mock_BookServiceClient struct {
	mock.Mock
}

func NewMock_BookServiceClient() *Mock_BookServiceClient {
	return &Mock_BookServiceClient{}
}

func (m *Mock_BookServiceClient) GetBook(ctx context.Context, req *bookv1.GetBookRequest) (*bookv1.BookResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*bookv1.BookResponse), args.Error(1)
}

func (m *Mock_BookServiceClient) ListBooks(ctx context.Context, req *bookv1.ListBooksRequest) (*bookv1.ListBooksResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*bookv1.ListBooksResponse), args.Error(1)
}

func (m *Mock_BookServiceClient) CreateBooks(ctx context.Context, req *bookv1.CreateBooksRequest) (*bookv1.CreateBooksResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*bookv1.CreateBooksResponse), args.Error(1)
}

func (m *Mock_BookServiceClient) UpdateBook(ctx context.Context, req *bookv1.UpdateBookRequest) (*bookv1.BookResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*bookv1.BookResponse), args.Error(1)
}

func (m *Mock_BookServiceClient) DeleteBooks(ctx context.Context, req *bookv1.DeleteBooksRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *Mock_BookServiceClient) CheckAvailability(ctx context.Context, req *bookv1.CheckAvailabilityRequest) (*bookv1.CheckAvailabilityResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*bookv1.CheckAvailabilityResponse), args.Error(1)
}

func (m *Mock_BookServiceClient) UpdateBookQuantity(ctx context.Context, req *bookv1.UpdateBookQuantityRequest) (*bookv1.UpdateBookQuantityResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*bookv1.UpdateBookQuantityResponse), args.Error(1)
}

func (m *Mock_BookServiceClient) GetConnection() *grpc.ClientConn {
	return nil
}

func newBookTestHandler(mc *Mock_BookServiceClient) *book_handler.BookHandler {
	return book_handler.NewBookHandlerWithClient(mc, testMapper, testLog)
}

/* GET BOOK */

func TestBookHandler_GetBook(t *testing.T) {
	tests := []struct {
		name           string
		uriID          string
		setupMock      func(*Mock_BookServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:  "GetBook — identifyTitle=true routes identifier to GetBookRequest_Title",
			uriID: "book-getbook-1",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("GetBook", mock.Anything,
					mock.MatchedBy(func(req *bookv1.GetBookRequest) bool {
						return req.GetTitle() == "book-getbook-1"
					}),
				).Return(&bookv1.BookResponse{
					Book: &bookv1.Book{Id: "uid-book-getbook-1", Title: "book-getbook-1"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "uid-book-getbook-1",
		},
		{
			name:  "GetBook response body contains book title",
			uriID: "book-getbook-2",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("GetBook", mock.Anything, mock.Anything).
					Return(&bookv1.BookResponse{
						Book: &bookv1.Book{Id: "uid-book-getbook-2", Title: "book-getbook-2"},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "book-getbook-2",
		},
		{
			name:           "Missing URI id (required) — ShouldBindUri fails and returns 400",
			uriID:          "",
			setupMock:      func(mc *Mock_BookServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request parameters",
		},
		{
			name:  "gRPC NotFound returns 404",
			uriID: "book-getbook-3",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("GetBook", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.NotFound, "Book not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Book not found",
		},
		{
			name:  "gRPC Internal returns 500",
			uriID: "book-getbook-4",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("GetBook", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Internal, "Internal server error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_BookServiceClient()
			tt.setupMock(mc)
			h := newBookTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/books/"+tt.uriID, nil)
			if tt.uriID != "" {
				c.Params = gin.Params{{Key: "id", Value: tt.uriID}}
			}

			h.GetBook(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* LIST BOOKS */

func TestBookHandler_ListBooks(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*Mock_BookServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "ListBooks — search_query and category query params bind correctly",
			queryParams: "?page=1&limit=10&search_query=book-listbooks-1&category=fiction",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("ListBooks", mock.Anything,
					mock.MatchedBy(func(req *bookv1.ListBooksRequest) bool {
						return req.GetSearchQuery() == "book-listbooks-1" &&
							req.GetCategory() == "fiction" &&
							req.GetPagination().GetPage() == 1 &&
							req.GetPagination().GetLimit() == 10
					}),
				).Return(&bookv1.ListBooksResponse{
					Books:      []*bookv1.Book{{Id: "uid-book-listbooks-1", Title: "book-listbooks-1"}},
					TotalCount: 1,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "uid-book-listbooks-1",
		},
		{
			name:        "ListBooks response returns all books and total_count",
			queryParams: "?page=1&limit=5&search_query=book-listbooks-2",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("ListBooks", mock.Anything, mock.Anything).
					Return(&bookv1.ListBooksResponse{
						Books: []*bookv1.Book{
							{Id: "uid-book-listbooks-2", Title: "book-listbooks-2"},
							{Id: "uid-book-listbooks-3", Title: "book-listbooks-3"},
						},
						TotalCount: 2,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_count":2`,
		},
		{
			name:        "Empty result — returns empty list and total_count 0",
			queryParams: "?page=1&limit=10",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("ListBooks", mock.Anything, mock.Anything).
					Return(&bookv1.ListBooksResponse{Books: nil, TotalCount: 0}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_count":0`,
		},
		{
			name:        "gRPC Internal returns 500",
			queryParams: "?page=1&limit=10",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("ListBooks", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Internal, "Internal server error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_BookServiceClient()
			tt.setupMock(mc)
			h := newBookTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/books"+tt.queryParams, nil)

			h.ListBooks(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* CREATE BOOKS */

func TestBookHandler_CreateBooks(t *testing.T) {
	tests := []struct {
		name           string
		inputBody      string
		setupMock      func(*Mock_BookServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "CreateBooks — books_payload binds to CreateBooksRequest correctly",
			inputBody: `{"books_payload":[{"title":"book-createbooks-1","author":"author-createbooks-1","isbn":"isbn-createbooks-1","quantity":5}]}`,
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("CreateBooks", mock.Anything,
					mock.MatchedBy(func(req *bookv1.CreateBooksRequest) bool {
						return len(req.GetBooks()) == 1 &&
							req.GetBooks()[0].GetTitle() == "book-createbooks-1" &&
							req.GetBooks()[0].GetAuthor() == "author-createbooks-1" &&
							req.GetBooks()[0].GetIsbn() == "isbn-createbooks-1" &&
							req.GetBooks()[0].GetTotalQuantity() == 5
					}),
				).Return(&bookv1.CreateBooksResponse{
					Books:        []*bookv1.Book{{Id: "uid-book-createbooks-1", Title: "book-createbooks-1"}},
					SuccessCount: 1,
				}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "uid-book-createbooks-1",
		},
		{
			name:      "CreateBooks response returns created_books and success_count",
			inputBody: `{"books_payload":[{"title":"book-createbooks-2","author":"author-createbooks-2","isbn":"isbn-createbooks-2","quantity":3},{"title":"book-createbooks-3","author":"author-createbooks-3","isbn":"isbn-createbooks-3","quantity":2}]}`,
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("CreateBooks", mock.Anything, mock.Anything).
					Return(&bookv1.CreateBooksResponse{
						Books: []*bookv1.Book{
							{Id: "uid-book-createbooks-2", Title: "book-createbooks-2"},
							{Id: "uid-book-createbooks-3", Title: "book-createbooks-3"},
						},
						SuccessCount: 2,
					}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `"success_count":2`,
		},
		{
			name:           "Missing books_payload (required) — ShouldBindJSON fails and returns 400",
			inputBody:      `{}`,
			setupMock:      func(mc *Mock_BookServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name:           "Invalid JSON — ShouldBindJSON fails and returns 400",
			inputBody:      `{not valid json`,
			setupMock:      func(mc *Mock_BookServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name:      "gRPC AlreadyExists returns 409",
			inputBody: `{"books_payload":[{"title":"book-createbooks-4","author":"author-createbooks-4","isbn":"isbn-createbooks-4","quantity":1}]}`,
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("CreateBooks", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.AlreadyExists, "Book already exists"))
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   "Book already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_BookServiceClient()
			tt.setupMock(mc)
			h := newBookTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/books", strings.NewReader(tt.inputBody))
			c.Request.Header.Set("Content-Type", "application/json")

			h.CreateBooks(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* UPDATE BOOK */

func TestBookHandler_UpdateBook(t *testing.T) {
	tests := []struct {
		name           string
		uriID          string
		inputBody      string
		setupMock      func(*Mock_BookServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "UpdateBook — URI id and body fields bind to UpdateBookRequest correctly",
			uriID:     "book-updatebook-1",
			inputBody: `{"title":"book-updatebook-1-updated","author":"author-updatebook-1"}`,
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("UpdateBook", mock.Anything,
					mock.MatchedBy(func(req *bookv1.UpdateBookRequest) bool {
						return req.GetId() == "book-updatebook-1" &&
							req.GetTitle() == "book-updatebook-1-updated" &&
							req.GetAuthor() == "author-updatebook-1"
					}),
				).Return(&bookv1.BookResponse{
					Book: &bookv1.Book{Id: "uid-book-updatebook-1", Title: "book-updatebook-1-updated"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "uid-book-updatebook-1",
		},
		{
			name:      "UpdateBook response returns updated book",
			uriID:     "book-updatebook-2",
			inputBody: `{"title":"book-updatebook-2-updated"}`,
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("UpdateBook", mock.Anything, mock.Anything).
					Return(&bookv1.BookResponse{
						Book: &bookv1.Book{Id: "uid-book-updatebook-2", Title: "book-updatebook-2-updated"},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "book-updatebook-2-updated",
		},
		{
			name:           "Missing URI id (required) — ShouldBindUri fails and returns 400",
			uriID:          "",
			inputBody:      `{"title":"update-title"}`,
			setupMock:      func(mc *Mock_BookServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request parameters",
		},
		{
			name:      "gRPC NotFound returns 404",
			uriID:     "book-updatebook-3",
			inputBody: `{"title":"book-updatebook-3-updated"}`,
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("UpdateBook", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.NotFound, "Book not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Book not found",
		},
		{
			name:      "gRPC Internal returns 500",
			uriID:     "book-updatebook-4",
			inputBody: `{"title":"book-updatebook-4-updated"}`,
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("UpdateBook", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Internal, "Internal server error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_BookServiceClient()
			tt.setupMock(mc)
			h := newBookTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPatch, "/v1/books/"+tt.uriID, strings.NewReader(tt.inputBody))
			c.Request.Header.Set("Content-Type", "application/json")
			if tt.uriID != "" {
				c.Params = gin.Params{{Key: "id", Value: tt.uriID}}
			}

			h.UpdateBook(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* DELETE BOOK */

func TestBookHandler_DeleteBook(t *testing.T) {
	tests := []struct {
		name           string
		paramID        string
		setupMock      func(*Mock_BookServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:    "DeleteBook — c.Param id binds to DeleteBooksRequest correctly",
			paramID: "book-deletebook-1",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("DeleteBooks", mock.Anything,
					mock.MatchedBy(func(req *bookv1.DeleteBooksRequest) bool {
						return len(req.GetIds()) == 1 && req.GetIds()[0] == "book-deletebook-1"
					}),
				).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		{
			name:           "Empty id — handler returns 400",
			paramID:        "",
			setupMock:      func(mc *Mock_BookServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Book ID is required",
		},
		{
			name:    "gRPC NotFound returns 404",
			paramID: "book-deletebook-2",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("DeleteBooks", mock.Anything, mock.Anything).
					Return(status.Error(codes.NotFound, "Book not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Book not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_BookServiceClient()
			tt.setupMock(mc)
			h := newBookTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodDelete, "/v1/books/"+tt.paramID, nil)
			if tt.paramID != "" {
				c.Params = gin.Params{{Key: "id", Value: tt.paramID}}
			}

			h.DeleteBook(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mc.AssertExpectations(t)
		})
	}
}

/* CHECK BOOK AVAILABILITY */

func TestBookHandler_CheckBookAvailability(t *testing.T) {
	tests := []struct {
		name           string
		uriID          string
		setupMock      func(*Mock_BookServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:  "CheckBookAvailability — URI id binds to BookID correctly",
			uriID: "book-checkavail-1",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("CheckAvailability", mock.Anything,
					mock.MatchedBy(func(req *bookv1.CheckAvailabilityRequest) bool {
						return req.GetBookId() == "book-checkavail-1"
					}),
				).Return(&bookv1.CheckAvailabilityResponse{
					IsAvailable:       true,
					AvailableQuantity: 5,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"is_available":true`,
		},
		{
			name:  "Book not available — response reflects is_available false",
			uriID: "book-checkavail-2",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("CheckAvailability", mock.Anything, mock.Anything).
					Return(&bookv1.CheckAvailabilityResponse{
						IsAvailable:       false,
						AvailableQuantity: 0,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"is_available":false`,
		},
		{
			name:           "Missing URI id (required) — ShouldBindUri fails and returns 400",
			uriID:          "",
			setupMock:      func(mc *Mock_BookServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request parameters",
		},
		{
			name:  "gRPC NotFound returns 404",
			uriID: "book-checkavail-3",
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("CheckAvailability", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.NotFound, "Book not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Book not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_BookServiceClient()
			tt.setupMock(mc)
			h := newBookTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/books/"+tt.uriID+"/availability", nil)
			if tt.uriID != "" {
				c.Params = gin.Params{{Key: "id", Value: tt.uriID}}
			}

			h.CheckBookAvailability(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* UPDATE BOOK QUANTITY */

func TestBookHandler_UpdateBookQuantity(t *testing.T) {
	tests := []struct {
		name           string
		uriID          string
		inputBody      string
		setupMock      func(*Mock_BookServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "UpdateBookQuantity — URI id and change_amount bind correctly",
			uriID:     "book-updateqty-1",
			inputBody: `{"change_amount": 5}`,
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("UpdateBookQuantity", mock.Anything,
					mock.MatchedBy(func(req *bookv1.UpdateBookQuantityRequest) bool {
						return req.GetBookId() == "book-updateqty-1" &&
							req.GetChangeAmount() == 5
					}),
				).Return(&bookv1.UpdateBookQuantityResponse{NewAvailableQuantity: 10}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"new_available_quantity":10`,
		},
		{
			name:      "UpdateBookQuantity with negative change_amount",
			uriID:     "book-updateqty-2",
			inputBody: `{"change_amount": -3}`,
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("UpdateBookQuantity", mock.Anything,
					mock.MatchedBy(func(req *bookv1.UpdateBookQuantityRequest) bool {
						return req.GetBookId() == "book-updateqty-2" &&
							req.GetChangeAmount() == -3
					}),
				).Return(&bookv1.UpdateBookQuantityResponse{NewAvailableQuantity: 2}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"new_available_quantity":2`,
		},
		{
			name:           "Missing URI id (required) — ShouldBindUri fails and returns 400",
			uriID:          "",
			inputBody:      `{"change_amount": 5}`,
			setupMock:      func(mc *Mock_BookServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request parameters",
		},
		{
			name:           "Missing change_amount (required) — ShouldBindJSON fails and returns 400",
			uriID:          "book-updateqty-3",
			inputBody:      `{}`,
			setupMock:      func(mc *Mock_BookServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name:      "gRPC NotFound returns 404",
			uriID:     "book-updateqty-4",
			inputBody: `{"change_amount": 1}`,
			setupMock: func(mc *Mock_BookServiceClient) {
				mc.On("UpdateBookQuantity", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.NotFound, "Book not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Book not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_BookServiceClient()
			tt.setupMock(mc)
			h := newBookTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPatch, "/v1/books/"+tt.uriID+"/quantity", strings.NewReader(tt.inputBody))
			c.Request.Header.Set("Content-Type", "application/json")
			if tt.uriID != "" {
				c.Params = gin.Params{{Key: "id", Value: tt.uriID}}
			}

			h.UpdateBookQuantity(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}
