package clients

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	orderv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/order"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientManager manages all service clients
type ClientManager struct {
	BookServiceClient  bookv1.BookServiceClient
	UserServiceClient  userv1.UserServiceClient
	OrderServiceClient orderv1.OrderServiceClient

	connections map[string]*grpc.ClientConn
	mu          sync.RWMutex
	logger      *logger.Logger
}

// ServiceAddresses contains addresses for all services
type ServiceAddresses struct {
	BookServiceAddr  string
	UserServiceAddr  string
	OrderServiceAddr string
}

// NewClientManager creates a new client manager and establishes connections
func NewClientManager(ctx context.Context, addrs ServiceAddresses, logger *logger.Logger) (*ClientManager, error) {
	cm := &ClientManager{
		connections: make(map[string]*grpc.ClientConn),
		logger:      logger,
	}

	// Create connections to each service
	connTimeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Connect to BookService
	bookConn, err := grpc.DialContext(
		connTimeoutCtx,
		addrs.BookServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*100)), // 100MB
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to book service: %w", err)
	}
	cm.connections["book"] = bookConn
	cm.BookServiceClient = bookv1.NewBookServiceClient(bookConn)

	// Connect to UserService
	userConn, err := grpc.DialContext(
		connTimeoutCtx,
		addrs.UserServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*100)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}
	cm.connections["user"] = userConn
	cm.UserServiceClient = userv1.NewUserServiceClient(userConn)

	// Connect to OrderService
	orderConn, err := grpc.DialContext(
		connTimeoutCtx,
		addrs.OrderServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*100)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to order service: %w", err)
	}
	cm.connections["order"] = orderConn
	cm.OrderServiceClient = orderv1.NewOrderServiceClient(orderConn)

	return cm, nil
}

// Close closes all service connections
func (cm *ClientManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for name, conn := range cm.connections {
		if err := conn.Close(); err != nil {
			cm.logger.Error(fmt.Sprintf("Failed to close %s connection", name), err, logger.Fields{})
			return err
		}
	}

	return nil
}

// HealthCheck checks the health of all service connections
func (cm *ClientManager) HealthCheck(ctx context.Context) map[string]bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	health := make(map[string]bool)

	for name, conn := range cm.connections {
		state := conn.GetState().String()
		health[name] = state == "READY"
	}

	return health
}

// RefreshToken calls the UserService RefreshToken RPC
func (cm *ClientManager) RefreshToken(ctx context.Context, userID string, refreshToken string) (*userv1.LoginResponse, error) {
	cm.logger.Info("RefreshToken called", logger.Fields{
		"user_id": userID,
	})

	req := &userv1.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	resp, err := cm.UserServiceClient.RefreshToken(ctx, req)
	if err != nil {
		cm.logger.Error("RefreshToken failed", err, logger.Fields{
			"user_id": userID,
		})
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return resp, nil
}
