package router

import (
	"context"
	"net/http"
	"time"

	book_httpv1 "github.com/DgHnG36/lib-management-system/services/gateway-service/internal/delivery/http/v1/book_handler"
	order_httpv1 "github.com/DgHnG36/lib-management-system/services/gateway-service/internal/delivery/http/v1/order_handler"
	user_httpv1 "github.com/DgHnG36/lib-management-system/services/gateway-service/internal/delivery/http/v1/user_handler"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/middleware"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func SetupRouter(
	corsMiddleware *middleware.CORSMiddleware,
	authMiddleware *middleware.AuthMiddleware,
	rateLimitMiddleware *middleware.RateLimitMiddleware,

	userHandler *user_httpv1.UserHandler,
	bookHandler *book_httpv1.BookHandler,
	orderHandler *order_httpv1.OrderHandler,

	logger *logger.Logger,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(withRequestID())
	router.Use(corsMiddleware.Handle())
	router.Use(withRequestLogger(logger))

	router.GET("/healthy", healthCheck)

	redisClient := rateLimitMiddleware.GetRedisClient()
	router.GET("/ready", readinessCheck(
		redisClient,
		userHandler,
		bookHandler,
		orderHandler,
		logger,
	))
	router.GET("/metrics", metricsCheck)

	v1 := router.Group("/api/v1")
	{
		v1.Use(rateLimitMiddleware.Handle())

		/* PUBLIC ROUTES */

		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
		}
		booksPublic := v1.Group("/books")
		{
			booksPublic.GET("", bookHandler.ListBooks)
			booksPublic.GET("/:id", bookHandler.GetBook)
		}

		/* PROTECTED ROUTES */
		v1.Use(authMiddleware.Handle())
		user := v1.Group("/user")
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PATCH("/profile", userHandler.UpdateProfile)
		}
		orders := v1.Group("/orders")
		{
			orders.POST("", orderHandler.CreateOrder)
			orders.GET("", orderHandler.ListMyOrders)
			orders.GET("/:id", orderHandler.GetOrder)
			orders.POST("/:id/cancel", orderHandler.CancelOrder)
		}

		/* MANAGER ROUTES */
		management := v1.Group("/management")
		management.Use(authMiddleware.RequireAdminOrManager())
		{
			users := management.Group("/users")
			{
				users.GET("", userHandler.ListUsers)
			}
			books := management.Group("/books")
			{
				books.POST("", bookHandler.CreateBooks)
				books.DELETE("/:id", bookHandler.DeleteBook)
				books.PUT("/:id", bookHandler.UpdateBook)
				books.PATCH("/:id/quantity", bookHandler.UpdateBookQuantity)
				books.GET("/:id/availability", bookHandler.CheckBookAvailability)
			}
			orders := management.Group("/orders")
			{
				orders.GET("", orderHandler.ListAllOrders)
				orders.PATCH("/:id/status", orderHandler.UpdateOrderStatus)
			}
		}

		/* ADMIN ROUTES */
		admin := v1.Group("/admin")
		admin.Use(authMiddleware.RequireAdmin())
		{
			admin.PATCH("/users/:id/vip", userHandler.UpdateVIPAccount)
			admin.DELETE("/users", userHandler.DeleteUsers)
		}
	}

	return router
}

func withRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.New().String()
		}

		c.Set("X-Request-ID", rid)
		c.Header("X-Request-ID", rid)
		c.Next()
	}
}

func withRequestLogger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		log.Info("HTTP Request", logger.Fields{
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       path,
			"latency":    latency,
			"ip":         c.ClientIP(),
			"request_id": c.GetString("X-Request-ID"),
			"user_id":    c.GetString("X-User-ID"),
		})
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func readinessCheck(redisClient *redis.Client,
	userHandler *user_httpv1.UserHandler,
	bookHandler *book_httpv1.BookHandler,
	orderHandler *order_httpv1.OrderHandler,
	logger *logger.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		// Check Redis connection
		if err := redisClient.Ping(ctx).Err(); err != nil {
			logger.Error("Readiness check failed: Redis is down", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"timestamp": time.Now().Format(time.RFC3339),
			})
			return
		}

		// Check other dependencies here
		if ok, err := userHandler.CheckConnection(); !ok {
			logger.Error("Readiness check failed: User service is down", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"timestamp": time.Now().Format(time.RFC3339),
			})
			return
		}

		if ok, err := bookHandler.CheckConnection(); !ok {
			logger.Error("Readiness check failed: Book service is down", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"timestamp": time.Now().Format(time.RFC3339),
			})
			return
		}

		if ok, err := orderHandler.CheckConnection(); !ok {
			logger.Error("Readiness check failed: Order service is down", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"timestamp": time.Now().Format(time.RFC3339),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "ready",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func metricsCheck(c *gin.Context) {
	gin.WrapH(promhttp.Handler())(c)
}
