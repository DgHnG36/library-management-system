package main

import (
	"context"
	"testing"

	"github.com/DgHnG36/lib-management-system/services/order-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/handlers"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/order-service/pkg/logger"
	commonv1 "github.com/DgHnG36/lib-management-system/shared/go/v1"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	orderv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/order"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newOrderHandlerForTest() (*handlers.OrderHandler, *MockOrderRepository, *MockBookServiceClient, *MockUserServiceClient) {
	repo := NewMockOrderRepository()
	bookClient := NewMockBookServiceClient()
	userClient := NewMockUserServiceClient()
	publisher := NewMockPublisher()
	orderSvc := applications.NewOrderService(repo, bookClient, userClient, publisher, logger.DefaultNewLogger())
	handler := handlers.NewOrderHandler(orderSvc, logger.DefaultNewLogger())
	return handler, repo, bookClient, userClient
}

func TestOrderHandler_CreateOrder_Success(t *testing.T) {
	handler, _, bookClient, _ := newOrderHandlerForTest()
	bookClient.books["book-1"] = &bookv1.Book{Id: "book-1", Title: "Test"}

	resp, err := handler.CreateOrder(context.Background(), &orderv1.CreateOrderRequest{
		UserId:     "user-1",
		BookIds:    []string{"book-1"},
		BorrowDays: 7,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Order)
	assert.Equal(t, orderv1.OrderStatus_PENDING, resp.Order.Status)
}

func TestOrderHandler_CreateOrder_InvalidInput(t *testing.T) {
	handler, _, _, _ := newOrderHandlerForTest()

	resp, err := handler.CreateOrder(context.Background(), &orderv1.CreateOrderRequest{
		UserId:     "",
		BookIds:    []string{"book-1"},
		BorrowDays: 7,
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestOrderHandler_GetOrder_Success(t *testing.T) {
	handler, repo, bookClient, userClient := newOrderHandlerForTest()
	repo.orders["order-1"] = &models.Order{
		ID:     "order-1",
		UserID: "user-1",
		Status: models.StatusPending,
		Books:  []models.OrderBook{{OrderID: "order-1", BookID: "book-1"}},
	}
	bookClient.books["book-1"] = &bookv1.Book{Id: "book-1", Title: "Test"}
	userClient.users["user-1"] = &userv1.User{Id: "user-1", Username: "u1"}

	resp, err := handler.GetOrder(context.Background(), &orderv1.GetOrderRequest{OrderId: "order-1"})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "order-1", resp.Order.Id)
}

func TestOrderHandler_GetOrder_MissingOrderID(t *testing.T) {
	handler, _, _, _ := newOrderHandlerForTest()

	resp, err := handler.GetOrder(context.Background(), &orderv1.GetOrderRequest{OrderId: ""})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestOrderHandler_ListMyOrders_Success(t *testing.T) {
	handler, repo, _, _ := newOrderHandlerForTest()
	repo.orders["1"] = &models.Order{ID: "1", UserID: "user-1", Status: models.StatusUnspecified}
	repo.orders["2"] = &models.Order{ID: "2", UserID: "user-1", Status: models.StatusUnspecified}

	resp, err := handler.ListMyOrders(context.Background(), &orderv1.ListMyOrdersRequest{
		UserId: "user-1",
		Pagination: &commonv1.PaginationRequest{
			Page:  1,
			Limit: 10,
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(2), resp.TotalCount)
}

func TestOrderHandler_ListMyOrders_MissingUserID(t *testing.T) {
	handler, _, _, _ := newOrderHandlerForTest()

	resp, err := handler.ListMyOrders(context.Background(), &orderv1.ListMyOrdersRequest{UserId: ""})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestOrderHandler_ListAllOrders_Success(t *testing.T) {
	handler, repo, _, _ := newOrderHandlerForTest()
	repo.orders["1"] = &models.Order{ID: "1", UserID: "user-1", Status: models.StatusUnspecified}
	repo.orders["2"] = &models.Order{ID: "2", UserID: "user-2", Status: models.StatusUnspecified}

	resp, err := handler.ListAllOrders(context.Background(), &orderv1.ListAllOrdersRequest{
		Pagination: &commonv1.PaginationRequest{Page: 1, Limit: 10},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(2), resp.TotalCount)
}

func TestOrderHandler_UpdateOrderStatus_Success(t *testing.T) {
	handler, repo, _, _ := newOrderHandlerForTest()
	repo.orders["order-1"] = &models.Order{ID: "order-1", UserID: "user-1", Status: models.StatusPending}

	resp, err := handler.UpdateOrderStatus(context.Background(), &orderv1.UpdateOrderStatusRequest{
		OrderId:   "order-1",
		NewStatus: orderv1.OrderStatus_APPROVED,
		Note:      "approved",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, orderv1.OrderStatus_APPROVED, resp.Order.Status)
}

func TestOrderHandler_UpdateOrderStatus_InvalidInput(t *testing.T) {
	handler, _, _, _ := newOrderHandlerForTest()

	resp, err := handler.UpdateOrderStatus(context.Background(), &orderv1.UpdateOrderStatusRequest{
		OrderId:   "",
		NewStatus: orderv1.OrderStatus_APPROVED,
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestOrderHandler_CancelOrder_Success(t *testing.T) {
	handler, repo, _, _ := newOrderHandlerForTest()
	repo.orders["order-1"] = &models.Order{ID: "order-1", UserID: "user-1", Status: models.StatusPending}

	resp, err := handler.CancelOrder(context.Background(), &orderv1.CancelOrderRequest{
		OrderId:      "order-1",
		UserId:       "user-1",
		CancelReason: "changed",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, orderv1.OrderStatus_CANCELED, resp.Order.Status)
}

func TestOrderHandler_CancelOrder_InvalidInput(t *testing.T) {
	handler, _, _, _ := newOrderHandlerForTest()

	resp, err := handler.CancelOrder(context.Background(), &orderv1.CancelOrderRequest{
		OrderId: "",
		UserId:  "user-1",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestOrderHandler_CancelOrder_FailedPrecondition(t *testing.T) {
	handler, repo, _, _ := newOrderHandlerForTest()
	repo.orders["order-1"] = &models.Order{ID: "order-1", UserID: "owner", Status: models.StatusPending}

	resp, err := handler.CancelOrder(context.Background(), &orderv1.CancelOrderRequest{
		OrderId:      "order-1",
		UserId:       "other",
		CancelReason: "changed",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}
