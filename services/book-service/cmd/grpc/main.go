package grpc

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DgHnG36/lib-management-system/services/book-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/config"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/handlers"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/repository"
	"github.com/DgHnG36/lib-management-system/services/book-service/pkg/interceptor"
	"github.com/DgHnG36/lib-management-system/services/book-service/pkg/logger"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.LoadFromEnv()

	// Initialize root logger
	logger.Init()
	svcLogger := logger.GetRootLogger()
	svcLogger.Info("Starting book-service", logger.Fields{
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
		"tables": []string{models.Book{}.TableName()},
	})

	// Wire up layers
	bookRepo := repository.NewBookRepo(db)
	bookService := applications.NewBookService(bookRepo, svcLogger)
	bookHandler := handlers.NewBookHandler(bookService, svcLogger)

	// Start gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.LoggingInterceptor(svcLogger),
			interceptor.RecoveryInterceptor(svcLogger),
		),
	)
	bookv1.RegisterBookServiceServer(grpcServer, bookHandler)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		svcLogger.Fatal("Failed to listen", err, logger.Fields{"addr": addr})
	}

	go func() {
		fmt.Printf(`
	 ____   ___   ___  _  __     ____  _____ ______     _____ ____ _____ 
	| __ ) / _ \ / _ \| |/ /    / ___|| ____|  _ \ \   / /_ _/ ___| ____|
	|  _ \| | | | | | | ' /____\___ \|  _| | |_) \ \ / / | | |   |  _|
	| |_) | |_| | |_| | . <_____|__) | |___|  _ < \ V /  | | |___| |___
	|____/ \___/ \___/|_|\_\   |____/|_____|_| \_\ \_/  |___\____|_____|
                    Book Service on port %d | %s | %s
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

	svcLogger.Info("Shutting down book-service...")
	grpcServer.GracefulStop()
	svcLogger.Info("Book service stopped")
}
