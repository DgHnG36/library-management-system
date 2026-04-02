package order_service_client

import (
	"context"
	"fmt"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	orderv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OrderServiceClient struct {
	client orderv1.OrderServiceClient
	conn   *grpc.ClientConn
	logger *logger.Logger
}

func NewOrderServiceClient(addr string, log *logger.Logger) (*OrderServiceClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(4*1024*1024)),
	)
	if err != nil {
		log.Error("Failed to connect to order service", err, logger.Fields{
			"address": addr,
		})
		return nil, fmt.Errorf("failed to connect to order service at %s: %w", addr, err)
	}
	log.Info("Successfully connected to order service", logger.Fields{
		"address": addr,
	})
	return &OrderServiceClient{
		client: orderv1.NewOrderServiceClient(conn),
		conn:   conn,
		logger: log,
	}, nil
}

func (oc *OrderServiceClient) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.OrderResponse, error) {
	oc.logger.Info("CreateOrder called to order service", logger.Fields{
		"user_id":     req.GetUserId(),
		"book_ids":    len(req.GetBookIds()),
		"borrow_days": req.GetBorrowDays(),
	})

	resp, err := oc.client.CreateOrder(ctx, req)
	if err != nil {
		oc.logger.Error("CreateOrder failed", err, logger.Fields{
			"user_id":     req.GetUserId(),
			"book_ids":    len(req.GetBookIds()),
			"borrow_days": req.GetBorrowDays(),
		})
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return resp, nil
}

func (oc *OrderServiceClient) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.OrderResponse, error) {
	oc.logger.Info("GetOrder called to order service", logger.Fields{
		"order_id": req.GetOrderId(),
	})

	resp, err := oc.client.GetOrder(ctx, req)
	if err != nil {
		oc.logger.Error("GetOrder failed", err, logger.Fields{
			"order_id": req.GetOrderId(),
		})
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return resp, nil
}

func (oc *OrderServiceClient) ListMyOrders(ctx context.Context, req *orderv1.ListMyOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	oc.logger.Info("ListMyOrders called to order service", logger.Fields{
		"user_id":       req.GetUserId(),
		"filter_status": req.GetFilterStatus(),
	})

	resp, err := oc.client.ListMyOrders(ctx, req)
	if err != nil {
		oc.logger.Error("ListMyOrders failed", err, logger.Fields{
			"user_id":       req.GetUserId(),
			"filter_status": req.GetFilterStatus(),
		})
		return nil, fmt.Errorf("failed to list my orders: %w", err)
	}

	return resp, nil
}

func (oc *OrderServiceClient) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.OrderResponse, error) {
	oc.logger.Info("CancelOrder called to order service", logger.Fields{
		"order_id":      req.GetOrderId(),
		"user_id":       req.GetUserId(),
		"cancel_reason": req.GetCancelReason(),
	})

	resp, err := oc.client.CancelOrder(ctx, req)
	if err != nil {
		oc.logger.Error("CancelOrder failed", err, logger.Fields{
			"order_id":      req.GetOrderId(),
			"user_id":       req.GetUserId(),
			"cancel_reason": req.GetCancelReason(),
		})
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return resp, nil
}

func (oc *OrderServiceClient) ListAllOrders(ctx context.Context, req *orderv1.ListAllOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	oc.logger.Info("ListAllOrders called to order service", logger.Fields{
		"filter_status":  req.GetFilterStatus(),
		"search_user_id": req.GetSearchUserId(),
	})

	resp, err := oc.client.ListAllOrders(ctx, req)
	if err != nil {
		oc.logger.Error("ListAllOrders failed", err, logger.Fields{
			"filter_status":  req.GetFilterStatus(),
			"search_user_id": req.GetSearchUserId(),
		})
		return nil, fmt.Errorf("failed to list all orders: %w", err)
	}

	return resp, nil
}

func (oc *OrderServiceClient) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.OrderResponse, error) {
	oc.logger.Info("UpdateOrderStatus called", logger.Fields{
		"order_id":   req.GetOrderId(),
		"new_status": req.GetNewStatus(),
		"note":       req.GetNote(),
	})

	resp, err := oc.client.UpdateOrderStatus(ctx, req)
	if err != nil {
		oc.logger.Error("UpdateOrderStatus failed", err, logger.Fields{
			"order_id":   req.GetOrderId(),
			"new_status": req.GetNewStatus(),
			"note":       req.GetNote(),
		})
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	return resp, nil
}

func (oc *OrderServiceClient) Close() error {
	if oc.conn != nil {
		return oc.conn.Close()
	}
	return nil
}

func (oc *OrderServiceClient) GetConnection() *grpc.ClientConn {
	return oc.conn
}
