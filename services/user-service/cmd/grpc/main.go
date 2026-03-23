package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DgHnG36/lib-management-system/services/user-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/config"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/handlers"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/repository"
	"github.com/DgHnG36/lib-management-system/services/user-service/pkg/interceptor"
	"github.com/DgHnG36/lib-management-system/services/user-service/pkg/logger"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.LoadFromEnv()

	// Initialize root logger
	logger.Init()
	svcLogger := logger.GetRootLogger()
	svcLogger.Info("Starting user-service", logger.Fields{
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
		"tables": []string{models.User{}.TableName()},
	})

	// Wire up layers
	userRepo := repository.NewUserRepo(db)
	userService := applications.NewUserService(userRepo, cfg.JWTSecret, cfg.JWTAlgorithm, cfg.JWTExpMins, svcLogger)
	userHandler := handlers.NewUserHandler(userService, svcLogger)

	// Start gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.LoggingInterceptor(svcLogger),
			interceptor.RecoveryInterceptor(svcLogger),
		),
	)
	userv1.RegisterUserServiceServer(grpcServer, userHandler)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		svcLogger.Fatal("Failed to listen", err, logger.Fields{"addr": addr})
	}

	go func() {
		fmt.Printf(`
	 _   _ ____  _____ ____      ____  _____ ______     _____ ____ _____ 
	| | | / ___|| ____|  _ \    / ___|| ____|  _ \ \   / /_ _/ ___| ____|
	| | | \___ \|  _| | |_) |___\___ \|  _| | |_) \ \ / / | | |   |  _|  
	| |_| |___) | |___|  _ <_____|__) | |___|  _ < \ V /  | | |___| |___ 
	 \___/|____/|_____|_| \_\   |____/|_____|_| \_\ \_/  |___\____|_____|
                    User Service on port %d | %s | %s
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

	svcLogger.Info("Shutting down user-service...")
	grpcServer.GracefulStop()
	svcLogger.Info("User service stopped")
}
