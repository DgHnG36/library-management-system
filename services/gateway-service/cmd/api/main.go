package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/clients/user_service_client"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/config"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/delivery/http/v1/book_handler"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/delivery/http/v1/order_handler"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/delivery/http/v1/user_handler"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/middleware"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/router"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/redis/go-redis/v9"
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
	rootLogger.Info("Logger initialized for gateway-service")

	// Initialize redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		rootLogger.Fatal("Failed to connect to Redis", err, logger.Fields{
			"addr": cfg.Redis.Addr,
			"db":   cfg.Redis.DB,
		})
	}

	rootLogger.Info("Successfully connected to Redis", logger.Fields{
		"addr": cfg.Redis.Addr,
		"db":   cfg.Redis.DB,
	})

	// Initialize middleware
	userSvcClient, err := user_service_client.NewUserServiceClient(cfg.GRPC.UserServiceAddr, rootLogger)
	if err != nil {
		rootLogger.Fatal("Failed to create user service client for JWT refresher", err, logger.Fields{
			"addr": cfg.GRPC.UserServiceAddr,
		})
	}
	jwtRefresher := user_service_client.NewJWTRefresher(userSvcClient, rootLogger)

	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret, cfg.JWT.Algorithm, cfg.JWT.ExpMins, jwtRefresher, rootLogger)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(redisClient, cfg.RateLimit.MaxRequests, cfg.RateLimit.WindowSeconds, rootLogger)
	corsMiddleware := middleware.NewCORSMiddleware(time.Duration(cfg.CORSMiddleware.MaxAge) * time.Second)

	// Initialize clients and handlers
	userHandler := user_handler.NewUserHandler(cfg.GRPC.UserServiceAddr, rootLogger)
	defer userHandler.Close()
	bookHandler := book_handler.NewBookHandler(cfg.GRPC.BookServiceAddr, rootLogger)
	defer bookHandler.Close()
	orderHandler := order_handler.NewOrderHandler(cfg.GRPC.OrderServiceAddr, rootLogger)
	defer orderHandler.Close()

	// Initialize router
	engine := router.SetupRouter(
		corsMiddleware,
		authMiddleware,
		rateLimitMiddleware,
		userHandler,
		bookHandler,
		orderHandler,
		rootLogger,
	)

	// Start server
	rootLogger.Info("Starting gateway-service...", logger.Fields{
		"host":    cfg.Server.Host,
		"port":    cfg.Server.Port,
		"env":     cfg.App.Environment,
		"version": cfg.App.Version,
	})
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		rootLogger.Info(fmt.Sprintf("Gateway service started on port %s", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			rootLogger.Fatal("Failed to start server", err)
		}
	}()

	fmt.Printf(`
	 _     __  __ ____         ____    _  _____ _______        ___ __   __
	| |   |  \/  / ___|       / ___|  / \|_   _| ____\ \      / / \\ \ / /
	| |   | |\/| \___ \ _____| |  _  / _ \ | | |  _|  \ \ /\ / / _ \\ V /
	| |___| |  | |___) |_____| |_| |/ ___ \| | | |___  \ V  V / ___ \| |
	|_____|_|  |_|____/       \____/_/   \_\_| |_____|  \_/\_/_/   \_\_|
						LMS Gateway Service on host %s:%s
						Environment: %s
						Version: %s
	`, cfg.Server.Host, cfg.Server.Port, cfg.App.Environment, cfg.App.Version)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	rootLogger.Info("Shutting down gateway-service...")

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		rootLogger.Fatal("Failed to gracefully shutdown server", err)
	}

	rootLogger.Info("Gateway service stopped", logger.Fields{
		"host":    cfg.Server.Host,
		"port":    cfg.Server.Port,
		"env":     cfg.App.Environment,
		"version": cfg.App.Version,
	})
	fmt.Printf(`
	  ____  ___   ___  ____    ______   _______ 
	 / ___|/ _ \ / _ \|  _ \  | __ ) \ / / ____|
	| |  _| | | | | | | | | | |  _ \\ V /|  _|
	| |_| | |_| | |_| | |_| | | |_) || | | |___
	 \____|\___/ \___/|____/  |____/ |_| |_____|

	`)
}
