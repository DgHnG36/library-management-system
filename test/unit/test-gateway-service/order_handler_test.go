package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/delivery/http/v1/order_handler"
	orderv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/order"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* MOCK ORDER SERVICE CLIENT — implements order_handler.OrderClientInterface */

type Mock_OrderServiceClient struct {
	mock.Mock
}

func NewMock_OrderServiceClient() *Mock_OrderServiceClient {
	return &Mock_OrderServiceClient{}
}

func (m *Mock_OrderServiceClient) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.OrderResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*orderv1.OrderResponse), args.Error(1)
}

func (m *Mock_OrderServiceClient) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.OrderResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*orderv1.OrderResponse), args.Error(1)
}

func (m *Mock_OrderServiceClient) ListMyOrders(ctx context.Context, req *orderv1.ListMyOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*orderv1.ListOrdersResponse), args.Error(1)
}

func (m *Mock_OrderServiceClient) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.OrderResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*orderv1.OrderResponse), args.Error(1)
}

func (m *Mock_OrderServiceClient) ListAllOrders(ctx context.Context, req *orderv1.ListAllOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*orderv1.ListOrdersResponse), args.Error(1)
}

func (m *Mock_OrderServiceClient) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.OrderResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*orderv1.OrderResponse), args.Error(1)
}

func (m *Mock_OrderServiceClient) GetConnection() *grpc.ClientConn {
	return nil
}

func newOrderTestHandler(mc *Mock_OrderServiceClient) *order_handler.OrderHandler {
	return order_handler.NewOrderHandlerWithClient(mc, testMapper, testLog)
}

/* CREATE ORDER */

func TestOrderHandler_CreateOrder(t *testing.T) {
	tests := []struct {
		name           string
		inputBody      string
		userID         string
		setupMock      func(*Mock_OrderServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "CreateOrder — book_ids, borrow_days, and X-User-ID bind to CreateOrderRequest correctly",
			inputBody: `{"book_ids":["book-createorder-1"],"borrow_days":7}`,
			userID:    "uid-user-createorder-1",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("CreateOrder", mock.Anything,
					mock.MatchedBy(func(req *orderv1.CreateOrderRequest) bool {
						return req.GetUserId() == "uid-user-createorder-1" &&
							len(req.GetBookIds()) == 1 &&
							req.GetBookIds()[0] == "book-createorder-1" &&
							req.GetBorrowDays() == 7
					}),
				).Return(&orderv1.OrderResponse{
					Order: &orderv1.Order{Id: "order-id-createorder-1"},
				}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "order-id-createorder-1",
		},
		{
			name:      "CreateOrder response returns order",
			inputBody: `{"book_ids":["book-createorder-2","book-createorder-3"],"borrow_days":14}`,
			userID:    "uid-user-createorder-2",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("CreateOrder", mock.Anything, mock.Anything).
					Return(&orderv1.OrderResponse{
						Order: &orderv1.Order{Id: "order-id-createorder-2"},
					}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "order-id-createorder-2",
		},
		{
			name:           "Missing book_ids (required) — ShouldBindJSON fails and returns 400",
			inputBody:      `{"borrow_days":7}`,
			userID:         "uid-user-createorder-3",
			setupMock:      func(mc *Mock_OrderServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name:           "Missing borrow_days (required) — ShouldBindJSON fails and returns 400",
			inputBody:      `{"book_ids":["book-createorder-4"]}`,
			userID:         "uid-user-createorder-4",
			setupMock:      func(mc *Mock_OrderServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name:      "gRPC NotFound returns 404",
			inputBody: `{"book_ids":["book-createorder-5"],"borrow_days":7}`,
			userID:    "uid-user-createorder-5",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("CreateOrder", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.NotFound, "Book not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Book not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_OrderServiceClient()
			tt.setupMock(mc)
			h := newOrderTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader(tt.inputBody))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("X-User-ID", tt.userID)

			h.CreateOrder(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* GET ORDER */

func TestOrderHandler_GetOrder(t *testing.T) {
	tests := []struct {
		name           string
		uriID          string
		setupMock      func(*Mock_OrderServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:  "GetOrder — URI id binds to OrderID correctly",
			uriID: "order-id-getorder-1",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("GetOrder", mock.Anything,
					mock.MatchedBy(func(req *orderv1.GetOrderRequest) bool {
						return req.GetOrderId() == "order-id-getorder-1"
					}),
				).Return(&orderv1.OrderResponse{
					Order: &orderv1.Order{Id: "order-id-getorder-1"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "order-id-getorder-1",
		},
		{
			name:  "GetOrder response returns order",
			uriID: "order-id-getorder-2",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("GetOrder", mock.Anything, mock.Anything).
					Return(&orderv1.OrderResponse{
						Order: &orderv1.Order{Id: "order-id-getorder-2"},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "order-id-getorder-2",
		},
		{
			name:           "Missing URI id (required) — ShouldBindUri fails and returns 400",
			uriID:          "",
			setupMock:      func(mc *Mock_OrderServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request parameters",
		},
		{
			name:  "gRPC NotFound returns 404",
			uriID: "order-id-getorder-3",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("GetOrder", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.NotFound, "Order not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Order not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_OrderServiceClient()
			tt.setupMock(mc)
			h := newOrderTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/orders/"+tt.uriID, nil)
			if tt.uriID != "" {
				c.Params = gin.Params{{Key: "id", Value: tt.uriID}}
			}

			h.GetOrder(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* LIST MY ORDERS */

func TestOrderHandler_ListMyOrders(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		userID         string
		setupMock      func(*Mock_OrderServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "ListMyOrders — query params and X-User-ID bind correctly",
			queryParams: "?page=1&limit=10&filter_status=pending",
			userID:      "uid-user-listmyorders-1",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("ListMyOrders", mock.Anything,
					mock.MatchedBy(func(req *orderv1.ListMyOrdersRequest) bool {
						return req.GetUserId() == "uid-user-listmyorders-1" &&
							req.GetPagination().GetPage() == 1 &&
							req.GetPagination().GetLimit() == 10
					}),
				).Return(&orderv1.ListOrdersResponse{
					Orders:     []*orderv1.Order{{Id: "order-id-listmyorders-1"}},
					TotalCount: 1,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "order-id-listmyorders-1",
		},
		{
			name:        "ListMyOrders response returns orders and total_count",
			queryParams: "?page=1&limit=5",
			userID:      "uid-user-listmyorders-2",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("ListMyOrders", mock.Anything, mock.Anything).
					Return(&orderv1.ListOrdersResponse{
						Orders: []*orderv1.Order{
							{Id: "order-id-listmyorders-2"},
							{Id: "order-id-listmyorders-3"},
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
			userID:      "uid-user-listmyorders-3",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("ListMyOrders", mock.Anything, mock.Anything).
					Return(&orderv1.ListOrdersResponse{Orders: nil, TotalCount: 0}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_count":0`,
		},
		{
			name:        "gRPC Internal returns 500",
			queryParams: "?page=1&limit=10",
			userID:      "uid-user-listmyorders-4",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("ListMyOrders", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Internal, "Internal server error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_OrderServiceClient()
			tt.setupMock(mc)
			h := newOrderTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/orders/my"+tt.queryParams, nil)
			c.Set("X-User-ID", tt.userID)

			h.ListMyOrders(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* CANCEL ORDER */

func TestOrderHandler_CancelOrder(t *testing.T) {
	tests := []struct {
		name           string
		uriID          string
		inputBody      string
		userID         string
		setupMock      func(*Mock_OrderServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "CancelOrder — URI id, cancel_reason, and X-User-ID bind correctly",
			uriID:     "order-id-cancelorder-1",
			inputBody: `{"cancel_reason":"not needed"}`,
			userID:    "uid-user-cancelorder-1",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("CancelOrder", mock.Anything,
					mock.MatchedBy(func(req *orderv1.CancelOrderRequest) bool {
						return req.GetOrderId() == "order-id-cancelorder-1" &&
							req.GetUserId() == "uid-user-cancelorder-1" &&
							req.GetCancelReason() == "not needed"
					}),
				).Return(&orderv1.OrderResponse{
					Order: &orderv1.Order{Id: "order-id-cancelorder-1"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "order-id-cancelorder-1",
		},
		{
			name:      "CancelOrder without reason — cancel_reason is optional",
			uriID:     "order-id-cancelorder-2",
			inputBody: "",
			userID:    "uid-user-cancelorder-2",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("CancelOrder", mock.Anything,
					mock.MatchedBy(func(req *orderv1.CancelOrderRequest) bool {
						return req.GetOrderId() == "order-id-cancelorder-2" &&
							req.GetUserId() == "uid-user-cancelorder-2" &&
							req.GetCancelReason() == ""
					}),
				).Return(&orderv1.OrderResponse{
					Order: &orderv1.Order{Id: "order-id-cancelorder-2"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "order-id-cancelorder-2",
		},
		{
			name:           "Missing URI id (required) — ShouldBindUri fails and returns 400",
			uriID:          "",
			inputBody:      "",
			userID:         "uid-user-cancelorder-3",
			setupMock:      func(mc *Mock_OrderServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request parameters",
		},
		{
			name:      "gRPC PermissionDenied returns 403",
			uriID:     "order-id-cancelorder-3",
			inputBody: `{"cancel_reason":"unauthorized cancel"}`,
			userID:    "uid-user-cancelorder-4",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("CancelOrder", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.PermissionDenied, "Permission denied"))
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_OrderServiceClient()
			tt.setupMock(mc)
			h := newOrderTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/orders/"+tt.uriID+"/cancel", strings.NewReader(tt.inputBody))
			if tt.inputBody != "" {
				c.Request.Header.Set("Content-Type", "application/json")
			}
			if tt.uriID != "" {
				c.Params = gin.Params{{Key: "id", Value: tt.uriID}}
			}
			c.Set("X-User-ID", tt.userID)

			h.CancelOrder(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* LIST ALL ORDERS */

func TestOrderHandler_ListAllOrders(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*Mock_OrderServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "ListAllOrders — filter_status and search_user_id query params bind correctly",
			queryParams: "?page=1&limit=10&filter_status=approved&search_user_id=uid-user-listall-1",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("ListAllOrders", mock.Anything,
					mock.MatchedBy(func(req *orderv1.ListAllOrdersRequest) bool {
						return req.GetSearchUserId() == "uid-user-listall-1" &&
							req.GetPagination().GetPage() == 1 &&
							req.GetPagination().GetLimit() == 10
					}),
				).Return(&orderv1.ListOrdersResponse{
					Orders:     []*orderv1.Order{{Id: "order-id-listall-1"}},
					TotalCount: 1,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "order-id-listall-1",
		},
		{
			name:        "ListAllOrders response returns orders and total_count",
			queryParams: "?page=1&limit=5",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("ListAllOrders", mock.Anything, mock.Anything).
					Return(&orderv1.ListOrdersResponse{
						Orders: []*orderv1.Order{
							{Id: "order-id-listall-2"},
							{Id: "order-id-listall-3"},
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
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("ListAllOrders", mock.Anything, mock.Anything).
					Return(&orderv1.ListOrdersResponse{Orders: nil, TotalCount: 0}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_count":0`,
		},
		{
			name:        "gRPC Internal returns 500",
			queryParams: "?page=1&limit=10",
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("ListAllOrders", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Internal, "Internal server error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_OrderServiceClient()
			tt.setupMock(mc)
			h := newOrderTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/orders"+tt.queryParams, nil)

			h.ListAllOrders(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* UPDATE ORDER STATUS */

func TestOrderHandler_UpdateOrderStatus(t *testing.T) {
	tests := []struct {
		name           string
		uriID          string
		inputBody      string
		setupMock      func(*Mock_OrderServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "UpdateOrderStatus — URI id and new_status bind to UpdateOrderStatusRequest correctly",
			uriID:     "order-id-updatestatus-1",
			inputBody: `{"new_status":"approved"}`,
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("UpdateOrderStatus", mock.Anything,
					mock.MatchedBy(func(req *orderv1.UpdateOrderStatusRequest) bool {
						return req.GetOrderId() == "order-id-updatestatus-1" &&
							req.GetNewStatus() == orderv1.OrderStatus_APPROVED
					}),
				).Return(&orderv1.OrderResponse{
					Order: &orderv1.Order{Id: "order-id-updatestatus-1"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "order-id-updatestatus-1",
		},
		{
			name:      "UpdateOrderStatus response returns updated order",
			uriID:     "order-id-updatestatus-2",
			inputBody: `{"new_status":"borrowed","note":"picked up"}`,
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("UpdateOrderStatus", mock.Anything,
					mock.MatchedBy(func(req *orderv1.UpdateOrderStatusRequest) bool {
						return req.GetOrderId() == "order-id-updatestatus-2" &&
							req.GetNewStatus() == orderv1.OrderStatus_BORROWED &&
							req.GetNote() == "picked up"
					}),
				).Return(&orderv1.OrderResponse{
					Order: &orderv1.Order{Id: "order-id-updatestatus-2"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "order-id-updatestatus-2",
		},
		{
			name:           "Missing URI id (required) — ShouldBindUri fails and returns 400",
			uriID:          "",
			inputBody:      `{"new_status":"approved"}`,
			setupMock:      func(mc *Mock_OrderServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request parameters",
		},
		{
			name:           "Missing new_status (required) — ShouldBindJSON fails and returns 400",
			uriID:          "order-id-updatestatus-3",
			inputBody:      `{}`,
			setupMock:      func(mc *Mock_OrderServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name:      "gRPC PermissionDenied returns 403",
			uriID:     "order-id-updatestatus-4",
			inputBody: `{"new_status":"canceled"}`,
			setupMock: func(mc *Mock_OrderServiceClient) {
				mc.On("UpdateOrderStatus", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.PermissionDenied, "Permission denied"))
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_OrderServiceClient()
			tt.setupMock(mc)
			h := newOrderTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPatch, "/v1/orders/"+tt.uriID+"/status", strings.NewReader(tt.inputBody))
			c.Request.Header.Set("Content-Type", "application/json")
			if tt.uriID != "" {
				c.Params = gin.Params{{Key: "id", Value: tt.uriID}}
			}

			h.UpdateOrderStatus(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}
