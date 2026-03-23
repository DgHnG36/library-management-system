package main

import (
	"context"
	"errors"
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

type MockOrderRepository struct {
	orders map[string]*models.Order
}

func NewMockOrderRepository() *MockOrderRepository {
	return &MockOrderRepository{orders: make(map[string]*models.Order)}
}

func (m *MockOrderRepository) Create(ctx context.Context, order *models.Order) error {
	if order == nil || order.ID == "" {
		return errors.New("order id is required")
	}
	m.orders[order.ID] = order
	return nil
}

func (m *MockOrderRepository) FindByID(ctx context.Context, orderID string) (*models.Order, error) {
	if orderID == "" {
		return nil, errors.New("order id is required")
	}
	if order, exists := m.orders[orderID]; exists {
		return order, nil
	}
	return nil, nil
}

func (m *MockOrderRepository) FindByUserID(ctx context.Context, userID string, page, limit int32, sortBy string, isDesc bool, filterStatus models.OrderStatus) ([]*models.Order, int32, error) {
	if userID == "" {
		return nil, 0, errors.New("user id is required")
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

func (m *MockOrderRepository) FindAll(ctx context.Context, page, limit int32, sortBy string, isDesc bool, filterStatus models.OrderStatus, searchUserID string) ([]*models.Order, int32, error) {
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

func (m *MockOrderRepository) UpdateStatus(ctx context.Context, orderID string, newStatus models.OrderStatus, note string) error {
	if order, ok := m.orders[orderID]; ok {
		order.Status = newStatus
		order.Note = note
		order.UpdatedAt = time.Now().UTC()
		return nil
	}
	return errors.New("order not found")
}

func (m *MockOrderRepository) UpdateReturnInfo(ctx context.Context, orderID string, penaltyAmount int32) error {
	if order, ok := m.orders[orderID]; ok {
		now := time.Now().UTC()
		order.ReturnDate = &now
		order.PenaltyAmount = penaltyAmount
		order.Status = models.StatusReturned
		order.UpdatedAt = now
		return nil
	}
	return errors.New("order not found")
}

func (m *MockOrderRepository) Cancel(ctx context.Context, orderID, userID, reason string) error {
	order, ok := m.orders[orderID]
	if !ok || order.UserID != userID || order.Status != models.StatusPending {
		return errors.New("cannot cancel")
	}
	order.Status = models.StatusCanceled
	order.CancelReason = reason
	order.UpdatedAt = time.Now().UTC()
	return nil
}

type MockBookServiceClient struct {
	books map[string]*bookv1.Book
}

func NewMockBookServiceClient() *MockBookServiceClient {
	return &MockBookServiceClient{books: make(map[string]*bookv1.Book)}
}

func (m *MockBookServiceClient) GetBook(ctx context.Context, in *bookv1.GetBookRequest, opts ...grpc.CallOption) (*bookv1.BookResponse, error) {
	bookID := in.GetId()
	if bookID == "" {
		return nil, status.Error(codes.InvalidArgument, "book id is required")
	}
	if book, exists := m.books[bookID]; exists {
		return &bookv1.BookResponse{Book: book}, nil
	}
	return nil, status.Error(codes.NotFound, "book not found")
}

func (m *MockBookServiceClient) ListBooks(ctx context.Context, in *bookv1.ListBooksRequest, opts ...grpc.CallOption) (*bookv1.ListBooksResponse, error) {
	books := make([]*bookv1.Book, 0, len(m.books))
	for _, book := range m.books {
		books = append(books, book)
	}
	return &bookv1.ListBooksResponse{Books: books, TotalCount: int32(len(books))}, nil
}

func (m *MockBookServiceClient) CreateBooks(ctx context.Context, in *bookv1.CreateBooksRequest, opts ...grpc.CallOption) (*bookv1.CreateBooksResponse, error) {
	return &bookv1.CreateBooksResponse{}, nil
}

func (m *MockBookServiceClient) UpdateBook(ctx context.Context, in *bookv1.UpdateBookRequest, opts ...grpc.CallOption) (*bookv1.BookResponse, error) {
	return &bookv1.BookResponse{}, nil
}

func (m *MockBookServiceClient) DeleteBooks(ctx context.Context, in *bookv1.DeleteBooksRequest, opts ...grpc.CallOption) (*commonv1.BaseResponse, error) {
	return &commonv1.BaseResponse{Status: 200}, nil
}

func (m *MockBookServiceClient) CheckAvailability(ctx context.Context, in *bookv1.CheckAvailabilityRequest, opts ...grpc.CallOption) (*bookv1.CheckAvailabilityResponse, error) {
	bookID := in.GetBookId()
	if _, exists := m.books[bookID]; !exists {
		return nil, status.Error(codes.NotFound, "book not found")
	}
	return &bookv1.CheckAvailabilityResponse{IsAvailable: true}, nil
}

func (m *MockBookServiceClient) UpdateBookQuantity(ctx context.Context, in *bookv1.UpdateBookQuantityRequest, opts ...grpc.CallOption) (*bookv1.UpdateBookQuantityResponse, error) {
	return &bookv1.UpdateBookQuantityResponse{}, nil
}

type MockUserServiceClient struct {
	users map[string]*userv1.User
}

func NewMockUserServiceClient() *MockUserServiceClient {
	return &MockUserServiceClient{users: make(map[string]*userv1.User)}
}

func (m *MockUserServiceClient) Register(ctx context.Context, in *userv1.RegisterRequest, opts ...grpc.CallOption) (*userv1.RegisterResponse, error) {
	return &userv1.RegisterResponse{}, nil
}

func (m *MockUserServiceClient) Login(ctx context.Context, in *userv1.LoginRequest, opts ...grpc.CallOption) (*userv1.LoginResponse, error) {
	return &userv1.LoginResponse{}, nil
}

func (m *MockUserServiceClient) GetProfile(ctx context.Context, in *userv1.GetProfileRequest, opts ...grpc.CallOption) (*userv1.UserProfileResponse, error) {
	userID := in.GetId()
	if user, exists := m.users[userID]; exists {
		return &userv1.UserProfileResponse{User: user}, nil
	}
	return nil, status.Error(codes.NotFound, "user not found")
}

func (m *MockUserServiceClient) UpdateProfile(ctx context.Context, in *userv1.UpdateProfileRequest, opts ...grpc.CallOption) (*userv1.UserProfileResponse, error) {
	return &userv1.UserProfileResponse{}, nil
}

func (m *MockUserServiceClient) UpdateVIPAccount(ctx context.Context, in *userv1.UpdateVIPAccountRequest, opts ...grpc.CallOption) (*userv1.UpdateVIPAccountResponse, error) {
	return &userv1.UpdateVIPAccountResponse{}, nil
}

func (m *MockUserServiceClient) ListUsers(ctx context.Context, in *userv1.ListUsersRequest, opts ...grpc.CallOption) (*userv1.ListUsersResponse, error) {
	return &userv1.ListUsersResponse{}, nil
}

func (m *MockUserServiceClient) DeleteUsers(ctx context.Context, in *userv1.DeleteUsersRequest, opts ...grpc.CallOption) (*commonv1.BaseResponse, error) {
	return &commonv1.BaseResponse{Status: 200}, nil
}

type MockPublisher struct{}

func NewMockPublisher() *MockPublisher { return &MockPublisher{} }

func (m *MockPublisher) Publish(routingKey string, payload map[string]interface{}) error { return nil }

func (m *MockPublisher) Close() {}

func newOrderServiceForTest() (*applications.OrderService, *MockOrderRepository, *MockBookServiceClient, *MockUserServiceClient) {
	repo := NewMockOrderRepository()
	bookClient := NewMockBookServiceClient()
	userClient := NewMockUserServiceClient()
	publisher := NewMockPublisher()
	service := applications.NewOrderService(repo, bookClient, userClient, publisher, logger.DefaultNewLogger())
	return service, repo, bookClient, userClient
}

func TestOrderService_CreateOrder_Success(t *testing.T) {
	service, _, bookClient, _ := newOrderServiceForTest()
	bookClient.books["book-1"] = &bookv1.Book{Id: "book-1", Title: "Test"}

	order, err := service.CreateOrder(context.Background(), "user-1", []string{"book-1"}, 7)

	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, "user-1", order.UserID)
	assert.Equal(t, models.StatusPending, order.Status)
	assert.Len(t, order.Books, 1)
}

func TestOrderService_CreateOrder_BookUnavailable(t *testing.T) {
	service, _, _, _ := newOrderServiceForTest()

	order, err := service.CreateOrder(context.Background(), "user-1", []string{"book-missing"}, 7)

	assert.Error(t, err)
	assert.Nil(t, order)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

func TestOrderService_GetOrder_Success(t *testing.T) {
	service, repo, bookClient, userClient := newOrderServiceForTest()

	repo.orders["order-1"] = &models.Order{
		ID:         "order-1",
		UserID:     "user-1",
		Status:     models.StatusPending,
		BorrowDate: time.Now().UTC(),
		DueDate:    time.Now().UTC().AddDate(0, 0, 7),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
		Books:      []models.OrderBook{{OrderID: "order-1", BookID: "book-1"}},
	}
	bookClient.books["book-1"] = &bookv1.Book{Id: "book-1", Title: "Book"}
	userClient.users["user-1"] = &userv1.User{Id: "user-1", Username: "u1"}

	order, user, books, err := service.GetOrder(context.Background(), "order-1")

	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.NotNil(t, user)
	assert.Len(t, books, 1)
}

func TestOrderService_GetOrder_NotFound(t *testing.T) {
	service, _, _, _ := newOrderServiceForTest()

	order, user, books, err := service.GetOrder(context.Background(), "missing")

	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Nil(t, user)
	assert.Nil(t, books)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestOrderService_ListMyOrders_Success(t *testing.T) {
	service, repo, _, _ := newOrderServiceForTest()
	repo.orders["1"] = &models.Order{ID: "1", UserID: "user-1", Status: models.StatusPending}
	repo.orders["2"] = &models.Order{ID: "2", UserID: "user-1", Status: models.StatusApproved}

	orders, total, err := service.ListMyOrders(context.Background(), "user-1", 1, 10, "", false, "")

	assert.NoError(t, err)
	assert.Equal(t, int32(2), total)
	assert.Len(t, orders, 2)
}

func TestOrderService_ListMyOrders_Filtered(t *testing.T) {
	service, repo, _, _ := newOrderServiceForTest()
	repo.orders["1"] = &models.Order{ID: "1", UserID: "user-1", Status: models.StatusPending}
	repo.orders["2"] = &models.Order{ID: "2", UserID: "user-1", Status: models.StatusApproved}

	orders, total, err := service.ListMyOrders(context.Background(), "user-1", 1, 10, "", false, models.StatusPending)

	assert.NoError(t, err)
	assert.Equal(t, int32(1), total)
	assert.Len(t, orders, 1)
	assert.Equal(t, models.StatusPending, orders[0].Status)
}

func TestOrderService_UpdateOrderStatus_Success(t *testing.T) {
	service, repo, _, _ := newOrderServiceForTest()
	repo.orders["order-1"] = &models.Order{ID: "order-1", UserID: "user-1", Status: models.StatusPending}

	updated, err := service.UpdateOrderStatus(context.Background(), "order-1", models.StatusApproved, "ok")

	assert.NoError(t, err)
	assert.NotNil(t, updated)
	assert.Equal(t, models.StatusApproved, updated.Status)
}

func TestOrderService_UpdateOrderStatus_NotFound(t *testing.T) {
	service, _, _, _ := newOrderServiceForTest()

	updated, err := service.UpdateOrderStatus(context.Background(), "missing", models.StatusApproved, "ok")

	assert.Error(t, err)
	assert.Nil(t, updated)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestOrderService_CancelOrder_Success(t *testing.T) {
	service, repo, _, _ := newOrderServiceForTest()
	repo.orders["order-1"] = &models.Order{ID: "order-1", UserID: "user-1", Status: models.StatusPending}

	order, err := service.CancelOrder(context.Background(), "order-1", "user-1", "changed")

	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, models.StatusCanceled, order.Status)
}

func TestOrderService_CancelOrder_FailedPrecondition(t *testing.T) {
	service, repo, _, _ := newOrderServiceForTest()
	repo.orders["order-1"] = &models.Order{ID: "order-1", UserID: "owner", Status: models.StatusPending}

	order, err := service.CancelOrder(context.Background(), "order-1", "other", "changed")

	assert.Error(t, err)
	assert.Nil(t, order)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}
