package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CORSMiddleware struct {
	allowedOrigins   []string
	allowedMethods   []string
	allowedHeaders   []string
	exposedHeaders   []string
	allowCredentials bool
	maxAge           time.Duration
}

func NewCORSMiddleware(allowedOrigins, allowedMethods, allowedHeaders, exposedHeaders []string, allowCredentials bool, maxAge time.Duration) *CORSMiddleware {
	return &CORSMiddleware{
		allowedOrigins:   allowedOrigins,
		allowedMethods:   allowedMethods,
		allowedHeaders:   allowedHeaders,
		exposedHeaders:   exposedHeaders,
		allowCredentials: allowCredentials,
		maxAge:           maxAge,
	}
}

func NewDefaultCORSMiddleware() *CORSMiddleware {
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:5173"} // Fix later after deploy to cloud server
	allowedMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	allowedHeaders := []string{"Origin", "Content-Type", "Content-Length", "Authorization",
		"Accept-Encoding", "X-CSRF-Token", "X-Requested-With", "X-Request-ID",
		"X-User-ID", "X-User-Role", "accept", "origin"}
	exposedHeaders := []string{"Content-Length", "Authorization", "X-Request-ID", "X-User-ID", "X-User-Role", "X-Rate-Limit-Remaining", "X-Rate-Limit-Limit"}
	allowCredentials := true
	maxAge := 12 * time.Hour

	return NewCORSMiddleware(allowedOrigins, allowedMethods, allowedHeaders, exposedHeaders, allowCredentials, maxAge)
}

func (m *CORSMiddleware) Handle() gin.HandlerFunc {
	allowMethods := strings.Join(m.allowedMethods, ", ")
	allowedHeaders := strings.Join(m.allowedHeaders, ", ")
	exposeHeaders := strings.Join(m.exposedHeaders, ", ")

	maxAgeStr := strconv.Itoa(int(m.maxAge.Seconds()))

	return func(c *gin.Context) {
		origin := c.GetHeader("origin")
		if origin != "" && m.isAllowOrigins(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", allowMethods)
			c.Header("Access-Control-Allow-Headers", allowedHeaders)
			c.Header("Access-Control-Expose-Headers", exposeHeaders)
			c.Header("Access-Control-Max-Age", maxAgeStr)
			if m.allowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func (m *CORSMiddleware) isAllowOrigins(origin string) bool {
	if origin == "" {
		return false
	}

	for _, allowedOrigin := range m.allowedOrigins {
		if m.matchOrigin(allowedOrigin, origin) {
			return true
		}
	}

	return false
}

func (m *CORSMiddleware) matchOrigin(allowedOrigin, origin string) bool {
	if (!m.allowCredentials && allowedOrigin == "*") || origin == allowedOrigin {
		return true
	}

	if strings.HasPrefix(allowedOrigin, "*.") {
		domain := strings.TrimPrefix(allowedOrigin, "*.")
		return strings.HasSuffix(origin, "."+domain)
	}

	return false
}
