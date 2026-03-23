package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/config"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/middleware"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/proxy"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/router"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.LoadFromEnv()

	// Initialize root logger service
	logger.Init()
	svcLogger := logger.GetRootLogger()
	svcLogger.Info("Logger gateway-service initialized")

	// Initialize redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPwd,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		svcLogger.Fatal("Failed to connect to Redis", err, logger.Fields{
			"addr": cfg.RedisAddr,
			"db":   cfg.RedisDB,
		})
	}

	svcLogger.Info("Successfully connected to Redis", logger.Fields{
		"addr": cfg.RedisAddr,
		"db":   cfg.RedisDB,
	})

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTSecret, cfg.JWTAlgorithm, svcLogger)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(redisClient, cfg.RateMaxRequests, cfg.RateWindow, svcLogger)
	corsMiddleware := middleware.NewDefaultCORSMiddleware()

	// Initialized reverse proxy
	reverseProxy := proxy.NewReverseProxy(cfg.TargetMap, svcLogger)

	// Initialize router
	engine := router.SetupRouter(authMiddleware, corsMiddleware, rateLimitMiddleware, reverseProxy, svcLogger)

	// Start server
	svcLogger.Info("Starting gateway-service...", logger.Fields{
		"port": cfg.Port,
	})
	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.Port),
		Handler:      engine,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	go func() {
		svcLogger.Info(fmt.Sprintf("Gateway service started on port %s", strconv.Itoa(cfg.Port)))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			svcLogger.Fatal("Failed to start server", err)
		}
	}()

	fmt.Printf(`
	 _     __  __ ____         ____    _  _____ _______        ___ __   __
	| |   |  \/  / ___|       / ___|  / \|_   _| ____\ \      / / \\ \ / /
	| |   | |\/| \___ \ _____| |  _  / _ \ | | |  _|  \ \ /\ / / _ \\ V /
	| |___| |  | |___) |_____| |_| |/ ___ \| | | |___  \ V  V / ___ \| |
	|_____|_|  |_|____/       \____/_/   \_\_| |_____|  \_/\_/_/   \_\_|
						LMS Gateway Service on port %d
						Environment: %s
						Version: %s
	`, cfg.Port, cfg.Environment, cfg.Version)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	svcLogger.Info("Shutting down gateway-service...")

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		svcLogger.Fatal("Failed to gracefully shutdown server", err)
	}

	svcLogger.Info("Gateway service stopped")

}
