package router

import (
	"net/http"
	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/middleware"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/proxy"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func SetupRouter(
	authMiddleware *middleware.AuthMiddleware,
	corsMiddleware *middleware.CORSMiddleware,
	rateLimitMiddleware *middleware.RateLimitMiddleware,
	reverseProxy *proxy.ReverseProxy,
	logger *logger.Logger,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(requestID())
	router.Use(requestLogger(logger))
	router.Use(corsMiddleware.Handle())

	router.GET("/healthy", healthCheck)
	router.GET("/ready", readinessCheck)
	router.GET("/metrics", metricsCheck)

	v1 := router.Group("/api/v1")
	{
		v1.Use(authMiddleware.Handle())
		v1.Use(rateLimitMiddleware.Handle())

		v1.Any("/*path", reverseProxy.Handle())

	}

	return router
}

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.New().String()
		}

		c.Header("X-Request-ID", rid)
		c.Next()
	}
}

func requestLogger(log *logger.Logger) gin.HandlerFunc {
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

func readinessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func metricsCheck(c *gin.Context) {
	handler := promhttp.Handler()
	handler.ServeHTTP(c.Writer, c.Request)
}
