package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/errors"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/gin-gonic/gin"
)

type ReverseProxy struct {
	targetMap map[string]string
	logger    *logger.Logger
}

func NewReverseProxy(targetMap map[string]string, logger *logger.Logger) *ReverseProxy {
	return &ReverseProxy{
		targetMap: targetMap,
		logger:    logger,
	}
}

func (rp *ReverseProxy) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		var targetHost string

		for prefix, host := range rp.targetMap {
			if strings.HasPrefix(path, prefix) {
				targetHost = host
				break
			}
		}

		if targetHost == "" {
			logger.Warn("Unavailable service")
			appErr := errors.ErrServiceUnavailable.Clone().WithMessage("service unavailable")
			c.AbortWithStatusJSON(int(appErr.HTTPStatus), appErr)
		}

		target, _ := url.Parse(targetHost)
		proxy := httputil.NewSingleHostReverseProxy(target)

		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.Host = target.Host
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/v1")

			if userID, exists := c.Get("X-User-ID"); exists {
				req.Header.Set("X-User-ID", userID.(string))
			}

			if role, exists := c.Get("X-User-Role"); exists {
				req.Header.Set("X-User-Role", role.(string))
			}

			if rid := c.GetString("request_id"); rid != "" {
				req.Header.Set("X-Request-ID", rid)
			}
		}

		proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			rp.logger.Error("Connect to service failed", err, logger.Fields{
				"path":   path,
				"target": targetHost,
			})

			appErr := errors.ErrServiceUnavailable.Clone().WithMessage("service unavailable")
			c.AbortWithStatusJSON(int(appErr.HTTPStatus), appErr)
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
