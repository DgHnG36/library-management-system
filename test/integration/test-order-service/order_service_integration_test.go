package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/order-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/handlers"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/repository"
	"github.com/DgHnG36/lib-management-system/services/order-service/pkg/logger"
	commonv1 "github.com/DgHnG36/lib-management-system/shared/go/v1"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	orderv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/order"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type fakeBookClient struct {
	db *gorm.DB
}

func (f *fakeBookClient) GetBook(ctx context.Context, in *bookv1.GetBookRequest, opts ...grpc.CallOption) (*bookv1.BookResponse, error) {
	id := in.GetId()
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "book id is required")
	}

	var row struct {
		ID    string
		Title string
	}
	err := f.db.WithContext(ctx).
		Table("books").
		Select("id, title").
		Where("id = ?", id).
		Take(&row).Error
	if err != nil {
		return nil, status.Error(codes.NotFound, "book not found")
	}

	return &bookv1.BookResponse{Book: &bookv1.Book{Id: row.ID, Title: row.Title}}, nil
}

func (f *fakeBookClient) ListBooks(ctx context.Context, in *bookv1.ListBooksRequest, opts ...grpc.CallOption) (*bookv1.ListBooksResponse, error) {
	return &bookv1.ListBooksResponse{}, nil
}

func (f *fakeBookClient) CreateBooks(ctx context.Context, in *bookv1.CreateBooksRequest, opts ...grpc.CallOption) (*bookv1.CreateBooksResponse, error) {
	return &bookv1.CreateBooksResponse{}, nil
}

func (f *fakeBookClient) UpdateBook(ctx context.Context, in *bookv1.UpdateBookRequest, opts ...grpc.CallOption) (*bookv1.BookResponse, error) {
	return &bookv1.BookResponse{}, nil
}

func (f *fakeBookClient) DeleteBooks(ctx context.Context, in *bookv1.DeleteBooksRequest, opts ...grpc.CallOption) (*commonv1.BaseResponse, error) {
	return &commonv1.BaseResponse{Status: 200}, nil
}

func (f *fakeBookClient) CheckAvailability(ctx context.Context, in *bookv1.CheckAvailabilityRequest, opts ...grpc.CallOption) (*bookv1.CheckAvailabilityResponse, error) {
	var row struct {
		ID                string
		AvailableQuantity int32
	}
	err := f.db.WithContext(ctx).
		Table("books").
		Select("id, available_quantity").
		Where("id = ?", in.GetBookId()).
		Take(&row).Error
	if err != nil || row.AvailableQuantity <= 0 {
		return nil, status.Error(codes.NotFound, "book not available")
	}

	return &bookv1.CheckAvailabilityResponse{IsAvailable: true, AvailableQuantity: row.AvailableQuantity}, nil
}

func (f *fakeBookClient) UpdateBookQuantity(ctx context.Context, in *bookv1.UpdateBookQuantityRequest, opts ...grpc.CallOption) (*bookv1.UpdateBookQuantityResponse, error) {
	result := f.db.WithContext(ctx).
		Table("books").
		Where("id = ?", in.GetBookId()).
		Update("available_quantity", gorm.Expr("available_quantity + ?", in.GetChangeAmount()))
	if result.Error != nil || result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "book not found")
	}

	var row struct {
		AvailableQuantity int32
	}
	err := f.db.WithContext(ctx).
		Table("books").
		Select("available_quantity").
		Where("id = ?", in.GetBookId()).
		Take(&row).Error
	if err != nil {
		return nil, status.Error(codes.Internal, "cannot load updated book quantity")
	}

	return &bookv1.UpdateBookQuantityResponse{Success: true, NewAvailableQuantity: row.AvailableQuantity}, nil
}

type fakeUserClient struct {
	db *gorm.DB
}

func (f *fakeUserClient) Register(ctx context.Context, in *userv1.RegisterRequest, opts ...grpc.CallOption) (*userv1.RegisterResponse, error) {
	return &userv1.RegisterResponse{}, nil
}

func (f *fakeUserClient) Login(ctx context.Context, in *userv1.LoginRequest, opts ...grpc.CallOption) (*userv1.LoginResponse, error) {
	return &userv1.LoginResponse{}, nil
}

func (f *fakeUserClient) GetProfile(ctx context.Context, in *userv1.GetProfileRequest, opts ...grpc.CallOption) (*userv1.UserProfileResponse, error) {
	var row struct {
		ID       string
		Username string
		Email    string
	}
	err := f.db.WithContext(ctx).
		Table("users").
		Select("id, username, email").
		Where("id = ?", in.GetId()).
		Take(&row).Error
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &userv1.UserProfileResponse{User: &userv1.User{Id: row.ID, Username: row.Username, Email: row.Email}}, nil
}

func (f *fakeUserClient) UpdateProfile(ctx context.Context, in *userv1.UpdateProfileRequest, opts ...grpc.CallOption) (*userv1.UserProfileResponse, error) {
	return &userv1.UserProfileResponse{}, nil
}

func (f *fakeUserClient) UpdateVIPAccount(ctx context.Context, in *userv1.UpdateVIPAccountRequest, opts ...grpc.CallOption) (*userv1.UpdateVIPAccountResponse, error) {
	return &userv1.UpdateVIPAccountResponse{}, nil
}

func (f *fakeUserClient) ListUsers(ctx context.Context, in *userv1.ListUsersRequest, opts ...grpc.CallOption) (*userv1.ListUsersResponse, error) {
	return &userv1.ListUsersResponse{}, nil
}

func (f *fakeUserClient) DeleteUsers(ctx context.Context, in *userv1.DeleteUsersRequest, opts ...grpc.CallOption) (*commonv1.BaseResponse, error) {
	return &commonv1.BaseResponse{Status: 200}, nil
}

type fakePublisher struct{}

func (f *fakePublisher) Publish(routingKey string, payload map[string]interface{}) error { return nil }
func (f *fakePublisher) Close()                                                          {}

type integrationIDs struct {
	PrimaryUserID   string
	SecondaryUserID string
	BookIDs         []string
}

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func openDBWithRetry(t *testing.T, host, port, dbName, user, password, label string) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", host, user, password, dbName, port)

	var db *gorm.DB
	var err error
	for i := 0; i < 5; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !assert.NoError(t, err, "failed to connect to %s database", label) {
		t.FailNow()
	}

	return db
}

func setupOrderTestDB(t *testing.T) *gorm.DB {
	host := getEnvOrDefault("TEST_ORDER_DB_HOST", "localhost")
	port := getEnvOrDefault("TEST_ORDER_DB_PORT", "15434")
	dbName := getEnvOrDefault("TEST_ORDER_DB_NAME", "order_db")
	user := getEnvOrDefault("TEST_ORDER_DB_USER", "postgres")
	password := getEnvOrDefault("TEST_ORDER_DB_PASSWORD", "postgres")

	db := openDBWithRetry(t, host, port, dbName, user, password, "order")

	err := db.AutoMigrate(&models.Order{}, &models.OrderBook{})
	if !assert.NoError(t, err, "failed to migrate order test database") {
		t.FailNow()
	}

	return db
}

func setupUserLookupDB(t *testing.T) *gorm.DB {
	host := getEnvOrDefault("TEST_USER_DB_HOST", "localhost")
	port := getEnvOrDefault("TEST_USER_DB_PORT", "15432")
	dbName := getEnvOrDefault("TEST_USER_DB_NAME", "user_db")
	user := getEnvOrDefault("TEST_USER_DB_USER", "postgres")
	password := getEnvOrDefault("TEST_USER_DB_PASSWORD", "postgres")

	return openDBWithRetry(t, host, port, dbName, user, password, "user")
}

func setupBookLookupDB(t *testing.T) *gorm.DB {
	host := getEnvOrDefault("TEST_BOOK_DB_HOST", "localhost")
	port := getEnvOrDefault("TEST_BOOK_DB_PORT", "15433")
	dbName := getEnvOrDefault("TEST_BOOK_DB_NAME", "book_db")
	user := getEnvOrDefault("TEST_BOOK_DB_USER", "postgres")
	password := getEnvOrDefault("TEST_BOOK_DB_PASSWORD", "postgres")

	return openDBWithRetry(t, host, port, dbName, user, password, "book")
}

func seedUser(t *testing.T, db *gorm.DB, prefix string) string {
	t.Helper()

	userID := uuid.New().String()
	username := fmt.Sprintf("%s_%s", prefix, uuid.NewString()[:8])
	email := fmt.Sprintf("%s_%s@test.local", prefix, uuid.NewString()[:8])
	now := time.Now().UTC()

	err := db.Exec(
		`INSERT INTO users (id, username, password, email, phone_number, role, is_vip, is_active, created_at, last_updated)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID,
		username,
		"integration-password",
		email,
		"+84000000000",
		"REGISTERED_USER",
		false,
		true,
		now,
		now,
	).Error
	if !assert.NoError(t, err, "failed to seed user data") {
		t.FailNow()
	}

	return userID
}

func seedBook(t *testing.T, db *gorm.DB, title, category string, totalQty int32) string {
	t.Helper()

	bookID := uuid.New().String()
	uniqueISBN := fmt.Sprintf("ISBN-%s", uuid.NewString()[:8])
	now := time.Now().UTC()

	err := db.Exec(
		`INSERT INTO books (id, title, author, isbn, category, description, total_quantity, available_quantity, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bookID,
		title,
		"integration-author",
		uniqueISBN,
		category,
		"integration-description",
		totalQty,
		totalQty,
		now,
		now,
	).Error
	if !assert.NoError(t, err, "failed to seed book data") {
		t.FailNow()
	}

	return bookID
}

func setupOrderService(t *testing.T) (*handlers.OrderHandler, integrationIDs) {
	db := setupOrderTestDB(t)
	if !assert.NoError(t, db.Exec("TRUNCATE TABLE order_books, orders RESTART IDENTITY CASCADE").Error) {
		t.FailNow()
	}

	userDB := setupUserLookupDB(t)
	bookDB := setupBookLookupDB(t)

	primaryUserID := seedUser(t, userDB, "order_it_primary")
	secondaryUserID := seedUser(t, userDB, "order_it_secondary")
	book1ID := seedBook(t, bookDB, "Integration Book 1", "tech", 3)
	book2ID := seedBook(t, bookDB, "Integration Book 2", "history", 2)

	t.Cleanup(func() {
		_ = userDB.Exec("DELETE FROM users WHERE id IN (?, ?)", primaryUserID, secondaryUserID).Error
		_ = bookDB.Exec("DELETE FROM books WHERE id IN (?, ?)", book1ID, book2ID).Error
	})

	orderRepo := repository.NewOrderRepo(db)
	bookClient := &fakeBookClient{db: bookDB}
	userClient := &fakeUserClient{db: userDB}
	publisher := &fakePublisher{}
	log := logger.DefaultNewLogger()

	orderService := applications.NewOrderService(orderRepo, bookClient, userClient, publisher, log)
	return handlers.NewOrderHandler(orderService, log), integrationIDs{
		PrimaryUserID:   primaryUserID,
		SecondaryUserID: secondaryUserID,
		BookIDs:         []string{book1ID, book2ID},
	}
}

func TestOrderService_CreateOrder_Success(t *testing.T) {
	handler, ids := setupOrderService(t)
	ctx := context.Background()

	resp, err := handler.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		UserId:     ids.PrimaryUserID,
		BookIds:    ids.BookIDs,
		BorrowDays: 7,
	})

	if !assert.NoError(t, err) {
		return
	}
	if !assert.NotNil(t, resp) || !assert.NotNil(t, resp.Order) {
		return
	}
	assert.Equal(t, orderv1.OrderStatus_PENDING, resp.Order.Status)
	assert.NotEmpty(t, resp.Order.Id)
}

func TestOrderService_CreateOrder_BookUnavailable(t *testing.T) {
	handler, ids := setupOrderService(t)
	ctx := context.Background()

	resp, err := handler.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		UserId:     ids.PrimaryUserID,
		BookIds:    []string{uuid.NewString()},
		BorrowDays: 5,
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

func TestOrderService_ListMyOrders_Success(t *testing.T) {
	handler, ids := setupOrderService(t)
	ctx := context.Background()

	_, _ = handler.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		UserId:     ids.PrimaryUserID,
		BookIds:    []string{ids.BookIDs[0]},
		BorrowDays: 3,
	})
	_, _ = handler.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		UserId:     ids.PrimaryUserID,
		BookIds:    []string{ids.BookIDs[1]},
		BorrowDays: 3,
	})

	resp, err := handler.ListMyOrders(ctx, &orderv1.ListMyOrdersRequest{
		UserId:       ids.PrimaryUserID,
		FilterStatus: orderv1.OrderStatus_PENDING,
		Pagination: &commonv1.PaginationRequest{
			Page:  1,
			Limit: 10,
		},
	})

	if !assert.NoError(t, err) {
		return
	}
	assert.NotNil(t, resp)
	assert.Equal(t, int32(2), resp.TotalCount)
}

func TestOrderService_ListAllOrders_FilterByUserAndStatus(t *testing.T) {
	handler, ids := setupOrderService(t)
	ctx := context.Background()

	_, _ = handler.CreateOrder(ctx, &orderv1.CreateOrderRequest{UserId: ids.PrimaryUserID, BookIds: []string{ids.BookIDs[0]}, BorrowDays: 2})
	_, _ = handler.CreateOrder(ctx, &orderv1.CreateOrderRequest{UserId: ids.SecondaryUserID, BookIds: []string{ids.BookIDs[1]}, BorrowDays: 2})

	resp, err := handler.ListAllOrders(ctx, &orderv1.ListAllOrdersRequest{
		Pagination:   &commonv1.PaginationRequest{Page: 1, Limit: 10},
		SearchUserId: ids.PrimaryUserID,
		FilterStatus: orderv1.OrderStatus_PENDING,
	})

	if !assert.NoError(t, err) {
		return
	}
	assert.NotNil(t, resp)
	assert.Equal(t, int32(1), resp.TotalCount)
}
