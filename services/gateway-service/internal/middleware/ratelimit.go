package middleware

import (
	"fmt"
	"strconv"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/errors"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const luaScript = `
local current = redis.call("INCR", KEYS[1])
if current == 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[1])
end

return current
`

type RateLimitMiddleware struct {
	redisClient   *redis.Client
	maxRequests   int32
	windowSeconds int32
	logger        *logger.Logger
}

func NewRateLimitMiddleware(redisClient *redis.Client, maxRequests, windowSeconds int32, logger *logger.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		redisClient:   redisClient,
		maxRequests:   maxRequests,
		windowSeconds: windowSeconds,
		logger:        logger,
	}
}

func (m *RateLimitMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := c.ClientIP()
		userRole := c.GetHeader("X-User-Role")

		if userRole == "admin" {
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%s:%s", userRole, identifier)
		ctx := c.Request.Context()

		result, err := m.redisClient.Eval(ctx, luaScript, []string{key}, m.windowSeconds).Result()
		if err != nil {
			m.logger.Error("Redis Lua Script error", errors.ErrInternalError.Clone().WithDetails(map[string]interface{}{
				"error": err.Error(),
			}))

			c.Next()
			return
		}

		count := result.(int64)
		remaining := int64(m.maxRequests) - count
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-Rate-Limit", strconv.Itoa(int(m.maxRequests)))
		c.Header("X-Rate-Limit-Remaining", strconv.FormatInt(remaining, 10))

		if count > int64(m.maxRequests) {
			appErr := errors.ErrRateLimitExceeded.Clone().WithDetails(map[string]interface{}{
				"limit":         m.maxRequests,
				"window":        fmt.Sprintf("%ds", m.windowSeconds),
				"current_count": count,
			})

			m.logger.Warn("Rate limit exceeded", logger.Fields{
				"ip":   identifier,
				"role": userRole,
				"code": appErr.Code,
			})

			c.AbortWithStatusJSON(int(appErr.HTTPStatus), appErr)

			return
		}

		c.Next()
	}
}
