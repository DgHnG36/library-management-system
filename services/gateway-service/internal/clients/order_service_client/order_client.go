package order_service_client

import (
	"context"
	"fmt"

	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	orderv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// OrderServiceClient wraps the generated OrderServiceClient with additional functionality
type OrderServiceClient struct {
	client orderv1.OrderServiceClient
	conn   *grpc.ClientConn
	logger *logger.Logger
}

// NewOrderServiceClient creates a new order service client connection
func NewOrderServiceClient(addr string, logger *logger.Logger) (*OrderServiceClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*100)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to order service at %s: %w", addr, err)
	}

	return &OrderServiceClient{
		client: orderv1.NewOrderServiceClient(conn),
		conn:   conn,
		logger: logger,
	}, nil
}

// CreateOrder creates a new order (borrow request)
func (oc *OrderServiceClient) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.OrderResponse, error) {
	oc.logger.Info("CreateOrder called", map[string]interface{}{
		"user_id":     req.UserId,
		"book_count":  len(req.BookIds),
		"borrow_days": req.BorrowDay,
	})

	resp, err := oc.client.CreateOrder(ctx, req)
	if err != nil {
		oc.logger.Error("CreateOrder failed", err, map[string]interface{}{
			"user_id":     req.UserId,
			"book_count":  len(req.BookIds),
			"borrow_days": req.BorrowDay,
		})
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return resp, nil
}

// GetOrder retrieves a specific order by ID
func (oc *OrderServiceClient) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.OrderResponse, error) {
	oc.logger.Info("GetOrder called", map[string]interface{}{
		"order_id": req.OrderId,
	})

	resp, err := oc.client.GetOrder(ctx, req)
	if err != nil {
		oc.logger.Error("GetOrder failed", err, map[string]interface{}{
			"order_id": req.OrderId,
		})
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return resp, nil
}

// ListMyOrders retrieves all orders for a specific user
func (oc *OrderServiceClient) ListMyOrders(ctx context.Context, req *orderv1.ListMyOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	oc.logger.Info("ListMyOrders called", map[string]interface{}{
		"user_id": req.UserId,
	})

	resp, err := oc.client.ListMyOrders(ctx, req)
	if err != nil {
		oc.logger.Error("ListMyOrders failed", err, map[string]interface{}{
			"user_id": req.UserId,
		})
		return nil, fmt.Errorf("failed to list my orders: %w", err)
	}

	return resp, nil
}

// CancelOrder cancels an existing order
func (oc *OrderServiceClient) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.OrderResponse, error) {
	oc.logger.Info("CancelOrder called", map[string]interface{}{
		"order_id": req.OrderId,
	})

	resp, err := oc.client.CancelOrder(ctx, req)
	if err != nil {
		oc.logger.Error("CancelOrder failed", err, map[string]interface{}{
			"order_id": req.OrderId,
		})
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return resp, nil
}

// ListAllOrders retrieves all orders with pagination (admin only)
func (oc *OrderServiceClient) ListAllOrders(ctx context.Context, req *orderv1.ListAllOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	oc.logger.Info("ListAllOrders called")

	resp, err := oc.client.ListAllOrders(ctx, req)
	if err != nil {
		oc.logger.Error("ListAllOrders failed", err)
		return nil, fmt.Errorf("failed to list all orders: %w", err)
	}

	return resp, nil
}

// UpdateOrderStatus updates the status of an order
func (oc *OrderServiceClient) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.OrderResponse, error) {
	oc.logger.Info("UpdateOrderStatus called", map[string]interface{}{
		"order_id":   req.OrderId,
		"new_status": req.NewStatus,
	})

	resp, err := oc.client.UpdateOrderStatus(ctx, req)
	if err != nil {
		oc.logger.Error("UpdateOrderStatus failed", err, map[string]interface{}{
			"order_id":   req.OrderId,
			"new_status": req.NewStatus,
		})
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	return resp, nil
}

// Close closes the connection to the order service
func (oc *OrderServiceClient) Close() error {
	if oc.conn != nil {
		return oc.conn.Close()
	}
	return nil
}

// GetConnection returns the underlying gRPC connection
func (oc *OrderServiceClient) GetConnection() *grpc.ClientConn {
	return oc.conn
}

// GetClient returns the underlying generated client
func (oc *OrderServiceClient) GetClient() orderv1.OrderServiceClient {
	return oc.client
}
