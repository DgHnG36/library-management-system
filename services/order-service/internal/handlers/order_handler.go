package handlers

import (
	"context"

	"github.com/DgHnG36/lib-management-system/services/order-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/order-service/pkg/logger"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	orderv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/order"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderHandler struct {
	orderv1.UnimplementedOrderServiceServer
	orderSvc *applications.OrderService
	logger   *logger.Logger
}

func NewOrderHandler(orderSvc *applications.OrderService, logger *logger.Logger) *OrderHandler {
	return &OrderHandler{
		orderSvc: orderSvc,
		logger:   logger,
	}
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.OrderResponse, error) {
	if req.UserId == "" || len(req.BookIds) == 0 || req.BorrowDays <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id, book_ids and borrow_days are required")
	}

	order, err := h.orderSvc.CreateOrder(ctx, req.GetUserId(), req.GetBookIds(), req.GetBorrowDays())
	if err != nil {
		h.logger.Error("Failed to create order", err)
		return nil, err
	}

	return &orderv1.OrderResponse{
		Order: toPbOrder(order, nil, nil),
	}, nil
}

func (h *OrderHandler) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.OrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "order_id is required")
	}

	order, user, books, err := h.orderSvc.GetOrder(ctx, req.GetOrderId())
	if err != nil {
		h.logger.Error("Failed to get order", err)
		return nil, err
	}

	return &orderv1.OrderResponse{
		Order: toPbOrder(order, user, books),
	}, nil
}

func (h *OrderHandler) ListMyOrders(ctx context.Context, req *orderv1.ListMyOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	var page, limit int32 = 1, 10
	var sortBy string
	var isDesc bool
	if req.GetPagination() != nil {
		page, limit = req.GetPagination().GetPage(), req.GetPagination().GetLimit()
		sortBy = req.GetPagination().GetSortBy()
		isDesc = req.GetPagination().GetIsDesc()
	}

	filterStatus := models.OrderStatus(req.FilterStatus.String())
	orders, total, err := h.orderSvc.ListMyOrders(ctx, req.GetUserId(), page, limit, sortBy, isDesc, filterStatus)
	if err != nil {
		h.logger.Error("Failed to list my orders", err)
		return nil, err
	}

	pbOrders := make([]*orderv1.Order, len(orders))
	for i, order := range orders {
		pbOrders[i] = toPbOrder(order, nil, nil)
	}

	return &orderv1.ListOrdersResponse{
		Orders:     pbOrders,
		TotalCount: total,
	}, nil
}

func (h *OrderHandler) ListAllOrders(ctx context.Context, req *orderv1.ListAllOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	var page, limit int32 = 1, 10
	var sortBy string
	var isDesc bool

	if req.GetPagination() != nil {
		page, limit = req.GetPagination().GetPage(), req.GetPagination().GetLimit()
		sortBy = req.GetPagination().GetSortBy()
		isDesc = req.GetPagination().GetIsDesc()
	}

	filterStatus := models.OrderStatus(req.FilterStatus.String())
	orders, total, err := h.orderSvc.ListAllOrders(ctx, page, limit, sortBy, isDesc, filterStatus, req.GetSearchUserId())
	if err != nil {
		h.logger.Error("Failed to list all orders", err)
		return nil, err
	}

	pbOrders := make([]*orderv1.Order, len(orders))
	for i, order := range orders {
		pbOrders[i] = toPbOrder(order, nil, nil)
	}

	return &orderv1.ListOrdersResponse{
		Orders:     pbOrders,
		TotalCount: total,
	}, nil
}

func (h *OrderHandler) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.OrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "order_id is required")
	}

	if req.GetNewStatus() == orderv1.OrderStatus_STATUS_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "new_status is required")
	}
	newStatus := models.OrderStatus(req.GetNewStatus().String())

	order, err := h.orderSvc.UpdateOrderStatus(ctx, req.GetOrderId(), newStatus, req.GetNote())
	if err != nil {
		h.logger.Error("Failed to update order status", err)
		return nil, err
	}

	return &orderv1.OrderResponse{
		Order: toPbOrder(order, nil, nil),
	}, nil
}

func (h *OrderHandler) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.OrderResponse, error) {
	if req.GetOrderId() == "" || req.GetUserId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "order_id and user_id are required")
	}

	order, err := h.orderSvc.CancelOrder(ctx, req.GetOrderId(), req.GetUserId(), req.GetCancelReason())
	if err != nil {
		h.logger.Error("Failed to cancel order", err)
		return nil, err
	}

	return &orderv1.OrderResponse{
		Order: toPbOrder(order, nil, nil),
	}, nil
}

/* HELPER METHODS */
func toPbOrder(order *models.Order, user *userv1.User, books []*bookv1.Book) *orderv1.Order {
	pbBooks := make([]*bookv1.Book, len(books))
	copy(pbBooks, books)

	pbOrder := &orderv1.Order{
		Id:            order.ID,
		Borrower:      user,
		BorrowedBooks: pbBooks,
		Status:        orderv1.OrderStatus(orderv1.OrderStatus_value[string(order.Status)]),
		BorrowDate:    timestamppb.New(order.BorrowDate),
		DueDate:       timestamppb.New(order.DueDate),
		PenaltyAmount: order.PenaltyAmount,
		Note:          order.Note,
		CreatedAt:     timestamppb.New(order.CreatedAt),
		UpdatedAt:     timestamppb.New(order.UpdatedAt),
	}

	if order.ReturnDate != nil {
		pbOrder.ReturnDate = timestamppb.New(*order.ReturnDate)
	}
	return pbOrder
}
