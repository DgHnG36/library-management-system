package grpc

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DgHnG36/lib-management-system/services/order-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/broker"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/config"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/handlers"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/repository"
	"github.com/DgHnG36/lib-management-system/services/order-service/pkg/interceptor"
	"github.com/DgHnG36/lib-management-system/services/order-service/pkg/logger"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	orderv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/order"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.LoadFromEnv()

	// Initialize service logger
	logger.Init()
	svcLogger := logger.GetRootLogger()
	svcLogger.Info("Starting order-service", logger.Fields{
		"env":     cfg.Environment,
		"version": cfg.Version,
		"port":    cfg.GRPCPort,
	})

	// Initialize DB
	dbCfg := &repository.DBConfig{
		Host:            cfg.DBHost,
		Port:            cfg.DBPort,
		User:            cfg.DBUser,
		Password:        cfg.DBPassword,
		DBName:          cfg.DBName,
		SSLMode:         cfg.DBSSLMode,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: time.Duration(cfg.DBConnMaxLifetime) * time.Minute,
	}

	db, err := repository.NewPostgresDB(dbCfg)
	if err != nil {
		svcLogger.Fatal("Failed to connect to database", err, logger.Fields{
			"host": cfg.DBHost,
			"name": cfg.DBName,
		})
	}
	svcLogger.Info("Connected to database", logger.Fields{"db": cfg.DBName})

	if err := repository.MigrateDB(db); err != nil {
		svcLogger.Fatal("Failed to migrate database", err)
	}
	svcLogger.Info("Database migrated", logger.Fields{
		"tables": []string{
			models.Order{}.TableName(),
			models.OrderBook{}.TableName(),
		},
	})

	// Initialize RabbitMQ connection
	publisher, err := broker.NewRabbitMQPublisher(
		cfg.RabbitMQURL,
		cfg.RabbitMQExchange,
		svcLogger,
	)
	if err != nil {
		svcLogger.Fatal("Failed to connect to RabbitMQ", err, logger.Fields{
			"url": cfg.RabbitMQURL,
		})
	}
	defer publisher.Close()
	svcLogger.Info("Connected to RabbitMQ", logger.Fields{"exchange": cfg.RabbitMQExchange})

	// Initialize other services gRPC clients
	userConn, err := grpc.NewClient(
		cfg.UserServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		svcLogger.Fatal("Failed to connect to user-service", err, logger.Fields{
			"addr": cfg.UserServiceAddr,
		})
	}
	defer userConn.Close()

	bookConn, err := grpc.NewClient(
		cfg.BookServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		svcLogger.Fatal("Failed to connect to book-service", err, logger.Fields{
			"addr": cfg.BookServiceAddr,
		})
	}
	defer bookConn.Close()

	userClient := userv1.NewUserServiceClient(userConn)
	bookClient := bookv1.NewBookServiceClient(bookConn)
	svcLogger.Info("gRPC clients initialized", logger.Fields{
		"user_service": cfg.UserServiceAddr,
		"book_service": cfg.BookServiceAddr,
	})

	orderRepo := repository.NewOrderRepo(db)
	orderService := applications.NewOrderService(orderRepo, bookClient, userClient, publisher, svcLogger)
	orderHandler := handlers.NewOrderHandler(orderService, svcLogger)

	// Start gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.LoggingInterceptor(svcLogger),
			interceptor.RecoveryInterceptor(svcLogger),
		),
	)
	orderv1.RegisterOrderServiceServer(grpcServer, orderHandler)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		svcLogger.Fatal("Failed to listen", err, logger.Fields{"addr": addr})
	}

	go func() {
		fmt.Printf(`
	 ___  ____  ____  _____ ____      ____  _____ ______     _____ ____ _____ 
 	/ _ \|  _ \|  _ \| ____|  _ \    / ___|| ____|  _ \ \   / /_ _/ ___| ____|
   | | | | |_) | | | |  _| | |_) |___\___ \|  _| | |_) \ \ / / | | |   |  _|
   | |_| |  _ <| |_| | |___|  _ <_____|__) | |___|  _ < \ V /  | | |___| |___
	\___/|_| \_\____/|_____|_| \_\   |____/|_____|_| \_\ \_/  |___\____|_____|
                    Order Service on port %d | %s | %s
`, cfg.GRPCPort, cfg.Environment, cfg.Version)

		svcLogger.Info("gRPC server started", logger.Fields{"addr": addr})
		if err := grpcServer.Serve(listener); err != nil {
			svcLogger.Fatal("gRPC server failed", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	svcLogger.Info("Shutting down order-service...")
	grpcServer.GracefulStop()
	svcLogger.Info("Order service stopped")
}
