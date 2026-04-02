package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

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
	// Initialize loader
	loader, err := config.NewConfigLoader()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize config loader: %v", err))
	}
	cfg := loader.GetConfig()

	// Initialize root logger
	logger.Init()
	rootLogger := logger.GetRootLogger()
	rootLogger.Info("Logger initialized for book-service")

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
		"db_name": cfg.Database.DBName,
	})

	if err := repository.MigrateDB(db); err != nil {
		rootLogger.Fatal("Failed to migrate database", err)
	}
	rootLogger.Info("Database migrated", logger.Fields{
		"tables": []string{models.Book{}.TableName()},
	})

	// Wire up layers
	bookRepo := repository.NewBookRepo(db)
	bookService := applications.NewBookService(bookRepo, rootLogger)
	bookHandler := handlers.NewBookHandler(bookService, rootLogger)

	// Start gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.MetadataInterceptor(),
			interceptor.LoggingInterceptor(rootLogger),
			interceptor.RecoveryInterceptor(rootLogger),
		),
	)
	bookv1.RegisterBookServiceServer(grpcServer, bookHandler)

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		rootLogger.Fatal("Failed to listen", err, logger.Fields{
			"addr": addr,
		})
	}

	go func() {
		fmt.Printf(`
	 ____   ___   ___  _  __     ____  _____ ______     _____ ____ _____ 
	| __ ) / _ \ / _ \| |/ /    / ___|| ____|  _ \ \   / /_ _/ ___| ____|
	|  _ \| | | | | | | ' /____\___ \|  _| | |_) \ \ / / | | |   |  _|
	| |_) | |_| | |_| | . <_____|__) | |___|  _ < \ V /  | | |___| |___
	|____/ \___/ \___/|_|\_\   |____/|_____|_| \_\ \_/  |___\____|_____|
                    LMS User Service on host %s:%s
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

	rootLogger.Info("Shutting down book-service...")
	grpcServer.GracefulStop()
	rootLogger.Info("Book service stopped", logger.Fields{
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
