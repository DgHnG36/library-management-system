package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

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
	// Initialize loader
	loader, err := config.NewConfigLoader()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize config loader: %v", err))
	}
	cfg := loader.GetConfig()

	// Initialize logger
	logger.Init()
	rootLogger := logger.GetRootLogger()
	rootLogger.Info("Logger initialized for order-service")

	// Initialize DB
	dbCfg := &repository.DBConfig{
		Host:            cfg.Database.DBHost,
		Port:            cfg.Database.DBPort,
		User:            cfg.Database.DBUser,
		Password:        cfg.Database.DBPwd,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.DBSSLMode,
		MaxOpenConns:    cfg.Database.DBMaxOpenConns,
		MaxIdleConns:    cfg.Database.DBMaxIdleConns,
		ConnMaxLifetime: cfg.Database.DBConnMaxLifetime,
	}

	db, err := repository.NewPostgresDB(dbCfg)
	if err != nil {
		rootLogger.Fatal("Failed to connect to database", err, logger.Fields{
			"addr":    fmt.Sprintf("%s:%s", cfg.Database.DBHost, cfg.Database.DBPort),
			"db_name": cfg.Database.DBName,
		})
	}
	rootLogger.Info("Connected to database", logger.Fields{
		"addr":    fmt.Sprintf("%s:%s", cfg.Database.DBHost, cfg.Database.DBPort),
		"db_name": cfg.Database.DBName,
	})

	if err := repository.MigrateDB(db); err != nil {
		rootLogger.Fatal("Failed to migrate database", err)
	}
	rootLogger.Info("Database migrated", logger.Fields{
		"tables": []string{
			models.Order{}.TableName(),
			models.OrderBook{}.TableName(),
		},
	})

	// Initialize message broker (rabbitmq for dev/test, sqs for prod)
	var publisher broker.Publisher
	switch cfg.BrokerType {
	case "sqs":
		publisher, err = broker.NewSQSPublisher(
			cfg.SQS.Region,
			cfg.SQS.QueueURL,
			cfg.SQS.AccessKeyID,
			cfg.SQS.SecretAccessKey,
			rootLogger,
		)
		if err != nil {
			rootLogger.Fatal("Failed to create SQS publisher", err, logger.Fields{
				"queue_url": cfg.SQS.QueueURL,
			})
		}
		rootLogger.Info("Connected to SQS", logger.Fields{"queue_url": cfg.SQS.QueueURL})
	default:
		publisher, err = broker.NewRabbitMQPublisher(
			cfg.RabbitMQ.URL,
			cfg.RabbitMQ.Exchange,
			rootLogger,
		)
		if err != nil {
			rootLogger.Fatal("Failed to connect to RabbitMQ", err, logger.Fields{
				"url": cfg.RabbitMQ.URL,
			})
		}
		rootLogger.Info("Connected to RabbitMQ", logger.Fields{"exchange": cfg.RabbitMQ.Exchange})
	}
	defer publisher.Close()

	// Initialize other services gRPC clients
	userConn, err := grpc.NewClient(
		cfg.Services.UserServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		rootLogger.Fatal("Failed to connect to user-service", err, logger.Fields{
			"addr": cfg.Services.UserServiceAddr,
		})
	}
	defer func() { _ = userConn.Close() }()

	bookConn, err := grpc.NewClient(
		cfg.Services.BookServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		rootLogger.Fatal("Failed to connect to book-service", err, logger.Fields{
			"addr": cfg.Services.BookServiceAddr,
		})
	}
	defer func() { _ = bookConn.Close() }()

	userClient := userv1.NewUserServiceClient(userConn)
	bookClient := bookv1.NewBookServiceClient(bookConn)
	rootLogger.Info("gRPC clients initialized", logger.Fields{
		"user_service": cfg.Services.UserServiceAddr,
		"book_service": cfg.Services.BookServiceAddr,
	})

	orderRepo := repository.NewOrderRepo(db)
	orderService := applications.NewOrderService(orderRepo, bookClient, userClient, publisher, rootLogger)
	orderHandler := handlers.NewOrderHandler(orderService, rootLogger)

	// Start gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.LoggingInterceptor(rootLogger),
			interceptor.RecoveryInterceptor(rootLogger),
		),
	)
	orderv1.RegisterOrderServiceServer(grpcServer, orderHandler)

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		rootLogger.Fatal("Failed to listen", err, logger.Fields{
			"addr": addr,
		})
	}

	go func() {
		fmt.Printf(`
	 ___  ____  ____  _____ ____      ____  _____ ______     _____ ____ _____ 
 	/ _ \|  _ \|  _ \| ____|  _ \    / ___|| ____|  _ \ \   / /_ _/ ___| ____|
   | | | | |_) | | | |  _| | |_) |___\___ \|  _| | |_) \ \ / / | | |   |  _|
   | |_| |  _ <| |_| | |___|  _ <_____|__) | |___|  _ < \ V /  | | |___| |___
	\___/|_| \_\____/|_____|_| \_\   |____/|_____|_| \_\ \_/  |___\____|_____|
                    LMS Order Service on host %s:%s
					Environment: %s
					Version: %s
`, cfg.Server.Host, cfg.Server.Port, cfg.App.Environment, cfg.App.Version)

		rootLogger.Info("gRPC server started", logger.Fields{
			"addr": addr,
		})
		if err := grpcServer.Serve(listener); err != nil {
			rootLogger.Fatal("gRPC server failed", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	rootLogger.Info("Shutting down order-service...")
	grpcServer.GracefulStop()
	rootLogger.Info("Order service stopped")
	fmt.Printf(`
	  ____  ___   ___  ____    ______   _______ 
	 / ___|/ _ \ / _ \|  _ \  | __ ) \ / / ____|
	| |  _| | | | | | | | | | |  _ \\ V /|  _|
	| |_| | |_| | |_| | |_| | | |_) || | | |___
	 \____|\___/ \___/|____/  |____/ |_| |_____|
	`)
}
