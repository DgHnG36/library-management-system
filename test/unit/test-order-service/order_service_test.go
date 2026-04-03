package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/order-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/order-service/pkg/logger"
	commonv1 "github.com/DgHnG36/lib-management-system/shared/go/v1"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* ─── Mock Repository ─────────────────────────────────────────────────────── */

/*
MockOrderRepository is an in-memory order store for unit tests.

	Shared across order_service_test.go and order_handler_test.go (same package).
*/
type MockOrderRepository struct {
	orders map[string]*models.Order
}

func NewMockOrderRepository() *MockOrderRepository {
	return &MockOrderRepository{orders: make(map[string]*models.Order)}
}

func (m *MockOrderRepository) Create(_ context.Context, order *models.Order) error {
	if order == nil || order.ID == "" {
		return status.Error(codes.InvalidArgument, "order id is required")
	}
	m.orders[order.ID] = order
	return nil
}

func (m *MockOrderRepository) FindByID(_ context.Context, orderID string) (*models.Order, error) {
	if orderID == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}
	if order, exists := m.orders[orderID]; exists {
		return order, nil
	}
	return nil, nil
}

func (m *MockOrderRepository) FindByUserID(_ context.Context, userID string, _, _ int32, _ string, _ bool, filterStatus models.OrderStatus) ([]*models.Order, int32, error) {
	if userID == "" {
		return nil, 0, status.Error(codes.InvalidArgument, "user id is required")
	}
	orders := make([]*models.Order, 0)
	for _, order := range m.orders {
		if order.UserID != userID {
			continue
		}
		if filterStatus != "" && order.Status != filterStatus {
			continue
		}
		orders = append(orders, order)
	}
	return orders, int32(len(orders)), nil
}

func (m *MockOrderRepository) FindAll(_ context.Context, _, _ int32, _ string, _ bool, filterStatus models.OrderStatus, searchUserID string) ([]*models.Order, int32, error) {
	orders := make([]*models.Order, 0)
	for _, order := range m.orders {
		if filterStatus != "" && order.Status != filterStatus {
			continue
		}
		if searchUserID != "" && order.UserID != searchUserID {
			continue
		}
		orders = append(orders, order)
	}
	return orders, int32(len(orders)), nil
}

func (m *MockOrderRepository) UpdateStatus(_ context.Context, orderID string, newStatus models.OrderStatus, note string) error {
	if order, ok := m.orders[orderID]; ok {
		order.Status = newStatus
		order.Note = note
		order.UpdatedAt = time.Now().UTC()
		return nil
	}
	return status.Error(codes.NotFound, "order not found")
}

func (m *MockOrderRepository) UpdateReturnInfo(_ context.Context, orderID string, penaltyAmount int32) error {
	if order, ok := m.orders[orderID]; ok {
		now := time.Now().UTC()
		order.ReturnDate = &now
		order.PenaltyAmount = penaltyAmount
		order.Status = models.StatusReturned
		order.UpdatedAt = now
		return nil
	}
	return status.Error(codes.NotFound, "order not found")
}

func (m *MockOrderRepository) Cancel(_ context.Context, orderID, userID, reason string) error {
	order, ok := m.orders[orderID]
	if !ok || order.UserID != userID || order.Status != models.StatusPending {
		return status.Error(codes.FailedPrecondition, "cannot cancel")
	}
	order.Status = models.StatusCanceled
	order.CancelReason = reason
	order.UpdatedAt = time.Now().UTC()
	return nil
}

/* ─── Mock gRPC Clients ───────────────────────────────────────────────────── */

type MockBookServiceClient struct {
	books map[string]*bookv1.Book
}

func NewMockBookServiceClient() *MockBookServiceClient {
	return &MockBookServiceClient{books: make(map[string]*bookv1.Book)}
}

func (m *MockBookServiceClient) GetBook(_ context.Context, in *bookv1.GetBookRequest, _ ...grpc.CallOption) (*bookv1.BookResponse, error) {
	if bookID := in.GetId(); bookID != "" {
		if book, exists := m.books[bookID]; exists {
			return &bookv1.BookResponse{Book: book}, nil
		}
		return nil, status.Error(codes.NotFound, "book not found")
	}
	return nil, status.Error(codes.InvalidArgument, "book id is required")
}

func (m *MockBookServiceClient) ListBooks(_ context.Context, _ *bookv1.ListBooksRequest, _ ...grpc.CallOption) (*bookv1.ListBooksResponse, error) {
	books := make([]*bookv1.Book, 0, len(m.books))
	for _, book := range m.books {
		books = append(books, book)
	}
	return &bookv1.ListBooksResponse{Books: books, TotalCount: int32(len(books))}, nil
}

func (m *MockBookServiceClient) CreateBooks(_ context.Context, _ *bookv1.CreateBooksRequest, _ ...grpc.CallOption) (*bookv1.CreateBooksResponse, error) {
	return &bookv1.CreateBooksResponse{}, nil
}

func (m *MockBookServiceClient) UpdateBook(_ context.Context, _ *bookv1.UpdateBookRequest, _ ...grpc.CallOption) (*bookv1.BookResponse, error) {
	return &bookv1.BookResponse{}, nil
}

func (m *MockBookServiceClient) DeleteBooks(_ context.Context, _ *bookv1.DeleteBooksRequest, _ ...grpc.CallOption) (*commonv1.BaseResponse, error) {
	return &commonv1.BaseResponse{Status: 200}, nil
}

func (m *MockBookServiceClient) CheckAvailability(_ context.Context, in *bookv1.CheckAvailabilityRequest, _ ...grpc.CallOption) (*bookv1.CheckAvailabilityResponse, error) {
	bookID := in.GetBookId()
	if _, exists := m.books[bookID]; !exists {
		return nil, status.Error(codes.NotFound, "book not found")
	}
	return &bookv1.CheckAvailabilityResponse{IsAvailable: true}, nil
}

func (m *MockBookServiceClient) UpdateBookQuantity(_ context.Context, _ *bookv1.UpdateBookQuantityRequest, _ ...grpc.CallOption) (*bookv1.UpdateBookQuantityResponse, error) {
	return &bookv1.UpdateBookQuantityResponse{}, nil
}

type MockUserServiceClient struct {
	users map[string]*userv1.User
	err   error
}

func NewMockUserServiceClient() *MockUserServiceClient {
	return &MockUserServiceClient{users: make(map[string]*userv1.User)}
}

func (m *MockUserServiceClient) Register(_ context.Context, _ *userv1.RegisterRequest, _ ...grpc.CallOption) (*userv1.RegisterResponse, error) {
	return &userv1.RegisterResponse{}, nil
}

func (m *MockUserServiceClient) Login(_ context.Context, _ *userv1.LoginRequest, _ ...grpc.CallOption) (*userv1.LoginResponse, error) {
	return &userv1.LoginResponse{}, nil
}

func (m *MockUserServiceClient) GetProfile(_ context.Context, in *userv1.GetProfileRequest, _ ...grpc.CallOption) (*userv1.UserProfileResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if user, exists := m.users[in.GetId()]; exists {
		return &userv1.UserProfileResponse{User: user}, nil
	}
	return nil, status.Error(codes.NotFound, "user not found")
}

func (m *MockUserServiceClient) UpdateProfile(_ context.Context, _ *userv1.UpdateProfileRequest, _ ...grpc.CallOption) (*userv1.UserProfileResponse, error) {
	return &userv1.UserProfileResponse{}, nil
}

func (m *MockUserServiceClient) UpdateVIPAccount(_ context.Context, _ *userv1.UpdateVIPAccountRequest, _ ...grpc.CallOption) (*userv1.UpdateVIPAccountResponse, error) {
	return &userv1.UpdateVIPAccountResponse{}, nil
}

func (m *MockUserServiceClient) ListUsers(_ context.Context, _ *userv1.ListUsersRequest, _ ...grpc.CallOption) (*userv1.ListUsersResponse, error) {
	return &userv1.ListUsersResponse{}, nil
}

func (m *MockUserServiceClient) DeleteUsers(_ context.Context, _ *userv1.DeleteUsersRequest, _ ...grpc.CallOption) (*commonv1.BaseResponse, error) {
	return &commonv1.BaseResponse{Status: 200}, nil
}

func (m *MockUserServiceClient) RefreshToken(_ context.Context, _ *userv1.RefreshTokenRequest, _ ...grpc.CallOption) (*userv1.LoginResponse, error) {
	return &userv1.LoginResponse{}, nil
}

type MockPublisher struct{}

func NewMockPublisher() *MockPublisher { return &MockPublisher{} }

func (m *MockPublisher) Publish(_ string, _ map[string]interface{}) error { return nil }

func (m *MockPublisher) Close() {}

/* ─── Shared test helpers ─────────────────────────────────────────────────── */

var testLog = logger.DefaultNewLogger()

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func newTestSvc() (*applications.OrderService, *MockOrderRepository, *MockBookServiceClient, *MockUserServiceClient) {
	repo := NewMockOrderRepository()
	bookClient := NewMockBookServiceClient()
	userClient := NewMockUserServiceClient()
	publisher := NewMockPublisher()
	svc := applications.NewOrderService(repo, bookClient, userClient, publisher, testLog)
	return svc, repo, bookClient, userClient
}

/* ─── TestOrderService_CreateOrder ───────────────────────────────────────── */
/* Verifies: order created with correct fields, book unavailable → FailedPrecondition. */

func TestOrderService_CreateOrder(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		bookIDs     []string
		borrowDays  int32
		setup       func(*MockBookServiceClient)
		wantErrCode codes.Code
		check       func(t *testing.T, o *models.Order)
	}{
		{
			name:       "success — order has correct userID, status PENDING, one book",
			userID:     "user-1",
			bookIDs:    []string{"book-1"},
			borrowDays: 7,
			setup: func(b *MockBookServiceClient) {
				b.books["book-1"] = &bookv1.Book{Id: "book-1", Title: "Test"}
			},
			check: func(t *testing.T, o *models.Order) {
				assert.NotNil(t, o)
				assert.Equal(t, "user-1", o.UserID)
				assert.Equal(t, models.StatusPending, o.Status)
				assert.Len(t, o.Books, 1)
			},
		},
		{
			name:        "book unavailable — FailedPrecondition",
			userID:      "user-1",
			bookIDs:     []string{"missing-book"},
			borrowDays:  7,
			wantErrCode: codes.FailedPrecondition,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, bookClient, _ := newTestSvc()
			if tt.setup != nil {
				tt.setup(bookClient)
			}
			o, err := svc.CreateOrder(context.Background(), tt.userID, tt.bookIDs, tt.borrowDays)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, o)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, o)
			}
		})
	}
}

/* ─── TestOrderService_GetOrder ───────────────────────────────────────────── */
/* Verifies: order+user+books returned for valid ID, not-found → NotFound,
   book service error → Internal, user service error → Internal. */

func TestOrderService_GetOrder(t *testing.T) {
	tests := []struct {
		name        string
		orderID     string
		setup       func(*MockOrderRepository, *MockBookServiceClient, *MockUserServiceClient)
		wantErrCode codes.Code
		check       func(t *testing.T, o *models.Order, u *userv1.User, books []*bookv1.Book)
	}{
		{
			name:    "success — order, user, and books all returned",
			orderID: "order-1",
			setup: func(r *MockOrderRepository, b *MockBookServiceClient, u *MockUserServiceClient) {
				r.orders["order-1"] = &models.Order{
					ID: "order-1", UserID: "user-1", Status: models.StatusPending,
					BorrowDate: time.Now().UTC(), DueDate: time.Now().UTC().AddDate(0, 0, 7),
					CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
					Books: []models.OrderBook{{OrderID: "order-1", BookID: "book-1"}},
				}
				b.books["book-1"] = &bookv1.Book{Id: "book-1", Title: "Book"}
				u.users["user-1"] = &userv1.User{Id: "user-1", Username: "u1"}
			},
			check: func(t *testing.T, o *models.Order, u *userv1.User, books []*bookv1.Book) {
				assert.NotNil(t, o)
				assert.NotNil(t, u)
				assert.Len(t, books, 1)
			},
		},
		{
			name:        "order not found — NotFound",
			orderID:     "missing",
			wantErrCode: codes.NotFound,
		},
		{
			name:    "book service error — Internal",
			orderID: "order-1",
			setup: func(r *MockOrderRepository, _ *MockBookServiceClient, u *MockUserServiceClient) {
				r.orders["order-1"] = &models.Order{
					ID: "order-1", UserID: "user-1", Status: models.StatusPending,
					Books: []models.OrderBook{{OrderID: "order-1", BookID: "book-missing"}},
				}
				u.users["user-1"] = &userv1.User{Id: "user-1", Username: "u1"}
			},
			wantErrCode: codes.Internal,
		},
		{
			name:    "user service error — Internal",
			orderID: "order-1",
			setup: func(r *MockOrderRepository, b *MockBookServiceClient, u *MockUserServiceClient) {
				r.orders["order-1"] = &models.Order{
					ID: "order-1", UserID: "user-1", Status: models.StatusPending,
					Books: []models.OrderBook{{OrderID: "order-1", BookID: "book-1"}},
				}
				b.books["book-1"] = &bookv1.Book{Id: "book-1", Title: "Book"}
				u.err = status.Error(codes.Internal, "user svc unavailable")
			},
			wantErrCode: codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, bookClient, userClient := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo, bookClient, userClient)
			}
			o, u, books, err := svc.GetOrder(context.Background(), tt.orderID)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, o)
				assert.Nil(t, u)
				assert.Nil(t, books)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, o, u, books)
			}
		})
	}
}

/* ─── TestOrderService_ListMyOrders ───────────────────────────────────────── */
/* Verifies: all user orders returned, filtered by status returns subset. */

func TestOrderService_ListMyOrders(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		filterStatus models.OrderStatus
		setup        func(*MockOrderRepository)
		wantErrCode  codes.Code
		check        func(t *testing.T, orders []*models.Order, total int32)
	}{
		{
			name:   "all orders for user — total 2, len 2",
			userID: "user-1",
			setup: func(r *MockOrderRepository) {
				r.orders["1"] = &models.Order{ID: "1", UserID: "user-1", Status: models.StatusPending}
				r.orders["2"] = &models.Order{ID: "2", UserID: "user-1", Status: models.StatusApproved}
			},
			check: func(t *testing.T, orders []*models.Order, total int32) {
				assert.Equal(t, int32(2), total)
				assert.Len(t, orders, 2)
			},
		},
		{
			name:         "filter by PENDING — only 1 result returned",
			userID:       "user-1",
			filterStatus: models.StatusPending,
			setup: func(r *MockOrderRepository) {
				r.orders["1"] = &models.Order{ID: "1", UserID: "user-1", Status: models.StatusPending}
				r.orders["2"] = &models.Order{ID: "2", UserID: "user-1", Status: models.StatusApproved}
			},
			check: func(t *testing.T, orders []*models.Order, total int32) {
				assert.Equal(t, int32(1), total)
				assert.Len(t, orders, 1)
				assert.Equal(t, models.StatusPending, orders[0].Status)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, _, _ := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			orders, total, err := svc.ListMyOrders(context.Background(), tt.userID, 1, 10, "", false, tt.filterStatus)
			if tt.wantErrCode != codes.OK {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, orders, total)
			}
		})
	}
}

/* ─── TestOrderService_ListAllOrders ──────────────────────────────────────── */
/* Verifies: filter by status + userID returns exact match. */

func TestOrderService_ListAllOrders(t *testing.T) {
	tests := []struct {
		name         string
		filterStatus models.OrderStatus
		searchUserID string
		setup        func(*MockOrderRepository)
		wantErrCode  codes.Code
		check        func(t *testing.T, orders []*models.Order, total int32)
	}{
		{
			name:         "filter by PENDING + user-1 — only order-1 returned",
			filterStatus: models.StatusPending,
			searchUserID: "user-1",
			setup: func(r *MockOrderRepository) {
				r.orders["1"] = &models.Order{ID: "1", UserID: "user-1", Status: models.StatusPending}
				r.orders["2"] = &models.Order{ID: "2", UserID: "user-1", Status: models.StatusApproved}
				r.orders["3"] = &models.Order{ID: "3", UserID: "user-2", Status: models.StatusPending}
			},
			check: func(t *testing.T, orders []*models.Order, total int32) {
				assert.Equal(t, int32(1), total)
				assert.Len(t, orders, 1)
				assert.Equal(t, "1", orders[0].ID)
			},
		},
		{
			name: "no filter — all 3 orders returned",
			setup: func(r *MockOrderRepository) {
				r.orders["1"] = &models.Order{ID: "1", UserID: "user-1", Status: models.StatusPending}
				r.orders["2"] = &models.Order{ID: "2", UserID: "user-1", Status: models.StatusApproved}
				r.orders["3"] = &models.Order{ID: "3", UserID: "user-2", Status: models.StatusPending}
			},
			check: func(t *testing.T, orders []*models.Order, total int32) {
				assert.Equal(t, int32(3), total)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, _, _ := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			orders, total, err := svc.ListAllOrders(context.Background(), 1, 10, "", false, tt.filterStatus, tt.searchUserID)
			if tt.wantErrCode != codes.OK {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, orders, total)
			}
		})
	}
}

/* ─── TestOrderService_UpdateOrderStatus ─────────────────────────────────── */
/* Verifies: status update reflected in returned order, returned status triggers
   penalty calculation, not-found → Internal. */

func TestOrderService_UpdateOrderStatus(t *testing.T) {
	tests := []struct {
		name        string
		orderID     string
		newStatus   models.OrderStatus
		note        string
		setup       func(*MockOrderRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, o *models.Order)
	}{
		{
			name:      "approve pending order — status APPROVED in returned order",
			orderID:   "order-1",
			newStatus: models.StatusApproved,
			note:      "approved",
			setup: func(r *MockOrderRepository) {
				r.orders["order-1"] = &models.Order{ID: "order-1", UserID: "user-1", Status: models.StatusPending}
			},
			check: func(t *testing.T, o *models.Order) {
				assert.Equal(t, models.StatusApproved, o.Status)
			},
		},
		{
			name:      "return overdue order — status RETURNED, penalty >= 5, ReturnDate set",
			orderID:   "order-1",
			newStatus: models.StatusReturned,
			note:      "returned",
			setup: func(r *MockOrderRepository) {
				dueDate := time.Now().UTC().AddDate(0, 0, -3)
				r.orders["order-1"] = &models.Order{
					ID: "order-1", UserID: "user-1", Status: models.StatusBorrowed,
					DueDate: dueDate,
					Books:   []models.OrderBook{{OrderID: "order-1", BookID: "book-1"}},
				}
			},
			check: func(t *testing.T, o *models.Order) {
				assert.Equal(t, models.StatusReturned, o.Status)
				assert.GreaterOrEqual(t, o.PenaltyAmount, int32(5))
				assert.NotNil(t, o.ReturnDate)
			},
		},
		{
			name:        "not found — Internal",
			orderID:     "missing",
			newStatus:   models.StatusApproved,
			wantErrCode: codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, _, _ := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			o, err := svc.UpdateOrderStatus(context.Background(), tt.orderID, tt.newStatus, tt.note)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, o)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, o)
			if tt.check != nil {
				tt.check(t, o)
			}
		})
	}
}

/* ─── TestOrderService_CancelOrder ───────────────────────────────────────── */
/* Verifies: owner can cancel pending order → status CANCELED, non-owner or
   non-pending → FailedPrecondition. */

func TestOrderService_CancelOrder(t *testing.T) {
	tests := []struct {
		name        string
		orderID     string
		userID      string
		reason      string
		setup       func(*MockOrderRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, o *models.Order)
	}{
		{
			name:    "owner cancels pending order — status CANCELED",
			orderID: "order-1",
			userID:  "user-1",
			reason:  "changed mind",
			setup: func(r *MockOrderRepository) {
				r.orders["order-1"] = &models.Order{ID: "order-1", UserID: "user-1", Status: models.StatusPending}
			},
			check: func(t *testing.T, o *models.Order) {
				assert.Equal(t, models.StatusCanceled, o.Status)
			},
		},
		{
			name:    "non-owner tries to cancel — FailedPrecondition",
			orderID: "order-1",
			userID:  "other",
			reason:  "reason",
			setup: func(r *MockOrderRepository) {
				r.orders["order-1"] = &models.Order{ID: "order-1", UserID: "owner", Status: models.StatusPending}
			},
			wantErrCode: codes.FailedPrecondition,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, _, _ := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			o, err := svc.CancelOrder(context.Background(), tt.orderID, tt.userID, tt.reason)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, o)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, o)
			if tt.check != nil {
				tt.check(t, o)
			}
		})
	}
}
