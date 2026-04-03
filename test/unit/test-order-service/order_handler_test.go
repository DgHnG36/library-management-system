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

/* ─── Handler test helpers ─────────────────────────────────────────────────── */

func newOrderHandlerForTest() (*handlers.OrderHandler, *MockOrderRepository, *MockBookServiceClient, *MockUserServiceClient) {
	repo := NewMockOrderRepository()
	bookClient := NewMockBookServiceClient()
	userClient := NewMockUserServiceClient()
	publisher := NewMockPublisher()
	orderSvc := applications.NewOrderService(repo, bookClient, userClient, publisher, logger.DefaultNewLogger())
	handler := handlers.NewOrderHandler(orderSvc, testLog)
	return handler, repo, bookClient, userClient
}

/* ─── TestOrderHandler_CreateOrder ───────────────────────────────────────── */
/* Verifies: order created with PENDING status, missing user_id or book_ids
   → InvalidArgument. */

func TestOrderHandler_CreateOrder(t *testing.T) {
	tests := []struct {
		name        string
		req         *orderv1.CreateOrderRequest
		setup       func(*MockBookServiceClient)
		wantErrCode codes.Code
		check       func(t *testing.T, r *orderv1.OrderResponse)
	}{
		{
			name: "valid request — order returned with status PENDING",
			req: &orderv1.CreateOrderRequest{
				UserId:     "user-1",
				BookIds:    []string{"book-1"},
				BorrowDays: 7,
			},
			setup: func(b *MockBookServiceClient) {
				b.books["book-1"] = &bookv1.Book{Id: "book-1", Title: "Test"}
			},
			check: func(t *testing.T, r *orderv1.OrderResponse) {
				assert.NotNil(t, r.GetOrder())
				assert.Equal(t, orderv1.OrderStatus_PENDING, r.GetOrder().GetStatus())
			},
		},
		{
			name: "missing user_id — InvalidArgument",
			req: &orderv1.CreateOrderRequest{
				UserId:     "",
				BookIds:    []string{"book-1"},
				BorrowDays: 7,
			},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "missing book_ids — InvalidArgument",
			req: &orderv1.CreateOrderRequest{
				UserId:     "user-1",
				BookIds:    []string{},
				BorrowDays: 7,
			},
			wantErrCode: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _, bookClient, _ := newOrderHandlerForTest()
			if tt.setup != nil {
				tt.setup(bookClient)
			}
			resp, err := handler.CreateOrder(context.Background(), tt.req)
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

/* ─── TestOrderHandler_GetOrder ───────────────────────────────────────────── */
/* Verifies: order ID in response matches request, empty order_id → InvalidArgument. */

func TestOrderHandler_GetOrder(t *testing.T) {
	tests := []struct {
		name        string
		req         *orderv1.GetOrderRequest
		setup       func(*MockOrderRepository, *MockBookServiceClient, *MockUserServiceClient)
		wantErrCode codes.Code
		check       func(t *testing.T, r *orderv1.OrderResponse)
	}{
		{
			name: "valid order_id — response order.Id matches",
			req:  &orderv1.GetOrderRequest{OrderId: "order-1"},
			setup: func(r *MockOrderRepository, b *MockBookServiceClient, u *MockUserServiceClient) {
				r.orders["order-1"] = &models.Order{
					ID:     "order-1",
					UserID: "user-1",
					Status: models.StatusPending,
					Books:  []models.OrderBook{{OrderID: "order-1", BookID: "book-1"}},
				}
				b.books["book-1"] = &bookv1.Book{Id: "book-1", Title: "Test"}
				u.users["user-1"] = &userv1.User{Id: "user-1", Username: "u1"}
			},
			check: func(t *testing.T, r *orderv1.OrderResponse) {
				assert.Equal(t, "order-1", r.GetOrder().GetId())
			},
		},
		{
			name:        "empty order_id — InvalidArgument",
			req:         &orderv1.GetOrderRequest{OrderId: ""},
			wantErrCode: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repo, bookClient, userClient := newOrderHandlerForTest()
			if tt.setup != nil {
				tt.setup(repo, bookClient, userClient)
			}
			resp, err := handler.GetOrder(context.Background(), tt.req)
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

/* ─── TestOrderHandler_ListMyOrders ───────────────────────────────────────── */
/* Verifies: TotalCount equals seeded orders for user, missing user_id
   → InvalidArgument. */

func TestOrderHandler_ListMyOrders(t *testing.T) {
	tests := []struct {
		name        string
		req         *orderv1.ListMyOrdersRequest
		setup       func(*MockOrderRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, r *orderv1.ListOrdersResponse)
	}{
		{
			name: "two orders for user — TotalCount=2",
			req: &orderv1.ListMyOrdersRequest{
				UserId:     "user-1",
				Pagination: &commonv1.PaginationRequest{Page: 1, Limit: 10},
			},
			setup: func(r *MockOrderRepository) {
				r.orders["1"] = &models.Order{ID: "1", UserID: "user-1", Status: models.StatusUnspecified}
				r.orders["2"] = &models.Order{ID: "2", UserID: "user-1", Status: models.StatusUnspecified}
			},
			check: func(t *testing.T, r *orderv1.ListOrdersResponse) {
				assert.Equal(t, int32(2), r.GetTotalCount())
			},
		},
		{
			name:        "missing user_id — InvalidArgument",
			req:         &orderv1.ListMyOrdersRequest{UserId: ""},
			wantErrCode: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repo, _, _ := newOrderHandlerForTest()
			if tt.setup != nil {
				tt.setup(repo)
			}
			resp, err := handler.ListMyOrders(context.Background(), tt.req)
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

/* ─── TestOrderHandler_ListAllOrders ──────────────────────────────────────── */
/* Verifies: TotalCount equals all seeded orders. */

func TestOrderHandler_ListAllOrders(t *testing.T) {
	tests := []struct {
		name        string
		req         *orderv1.ListAllOrdersRequest
		setup       func(*MockOrderRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, r *orderv1.ListOrdersResponse)
	}{
		{
			name: "two orders seeded — TotalCount=2",
			req:  &orderv1.ListAllOrdersRequest{Pagination: &commonv1.PaginationRequest{Page: 1, Limit: 10}},
			setup: func(r *MockOrderRepository) {
				r.orders["1"] = &models.Order{ID: "1", UserID: "user-1", Status: models.StatusUnspecified}
				r.orders["2"] = &models.Order{ID: "2", UserID: "user-2", Status: models.StatusUnspecified}
			},
			check: func(t *testing.T, r *orderv1.ListOrdersResponse) {
				assert.Equal(t, int32(2), r.GetTotalCount())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repo, _, _ := newOrderHandlerForTest()
			if tt.setup != nil {
				tt.setup(repo)
			}
			resp, err := handler.ListAllOrders(context.Background(), tt.req)
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

/* ─── TestOrderHandler_UpdateOrderStatus ─────────────────────────────────── */
/* Verifies: updated status reflected in response, empty order_id or unspecified
   status → InvalidArgument. */

func TestOrderHandler_UpdateOrderStatus(t *testing.T) {
	tests := []struct {
		name        string
		req         *orderv1.UpdateOrderStatusRequest
		setup       func(*MockOrderRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, r *orderv1.OrderResponse)
	}{
		{
			name: "valid request — response order.Status equals APPROVED",
			req: &orderv1.UpdateOrderStatusRequest{
				OrderId:   "order-1",
				NewStatus: orderv1.OrderStatus_APPROVED,
				Note:      "approved",
			},
			setup: func(r *MockOrderRepository) {
				r.orders["order-1"] = &models.Order{ID: "order-1", UserID: "user-1", Status: models.StatusPending}
			},
			check: func(t *testing.T, r *orderv1.OrderResponse) {
				assert.Equal(t, orderv1.OrderStatus_APPROVED, r.GetOrder().GetStatus())
			},
		},
		{
			name: "empty order_id — InvalidArgument",
			req: &orderv1.UpdateOrderStatusRequest{
				OrderId:   "",
				NewStatus: orderv1.OrderStatus_APPROVED,
			},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "unspecified new_status — InvalidArgument",
			req: &orderv1.UpdateOrderStatusRequest{
				OrderId:   "order-1",
				NewStatus: orderv1.OrderStatus_STATUS_UNSPECIFIED,
			},
			wantErrCode: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repo, _, _ := newOrderHandlerForTest()
			if tt.setup != nil {
				tt.setup(repo)
			}
			resp, err := handler.UpdateOrderStatus(context.Background(), tt.req)
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

/* ─── TestOrderHandler_CancelOrder ───────────────────────────────────────── */
/* Verifies: owner cancels pending order → status CANCELED, empty order_id or
   user_id → InvalidArgument, non-owner → FailedPrecondition. */

func TestOrderHandler_CancelOrder(t *testing.T) {
	tests := []struct {
		name        string
		req         *orderv1.CancelOrderRequest
		setup       func(*MockOrderRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, r *orderv1.OrderResponse)
	}{
		{
			name: "valid request — order status CANCELED in response",
			req: &orderv1.CancelOrderRequest{
				OrderId:      "order-1",
				UserId:       "user-1",
				CancelReason: "changed mind",
			},
			setup: func(r *MockOrderRepository) {
				r.orders["order-1"] = &models.Order{ID: "order-1", UserID: "user-1", Status: models.StatusPending}
			},
			check: func(t *testing.T, r *orderv1.OrderResponse) {
				assert.Equal(t, orderv1.OrderStatus_CANCELED, r.GetOrder().GetStatus())
			},
		},
		{
			name: "empty order_id — InvalidArgument",
			req: &orderv1.CancelOrderRequest{
				OrderId: "",
				UserId:  "user-1",
			},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "non-owner tries to cancel — FailedPrecondition",
			req: &orderv1.CancelOrderRequest{
				OrderId:      "order-1",
				UserId:       "other",
				CancelReason: "reason",
			},
			setup: func(r *MockOrderRepository) {
				r.orders["order-1"] = &models.Order{ID: "order-1", UserID: "owner", Status: models.StatusPending}
			},
			wantErrCode: codes.FailedPrecondition,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repo, _, _ := newOrderHandlerForTest()
			if tt.setup != nil {
				tt.setup(repo)
			}
			resp, err := handler.CancelOrder(context.Background(), tt.req)
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

