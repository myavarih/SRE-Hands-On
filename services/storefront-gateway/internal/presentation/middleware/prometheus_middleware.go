package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/myavarih/sre-hands-on/pkg/telemetry"
)

func PrometheusMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		handler := c.FullPath()
		if handler == "" {
			handler = "unknown"
		}
		method := c.Request.Method

		telemetry.HTTPRequestsTotal.WithLabelValues(serviceName, handler, method, status).Inc()
		telemetry.HTTPRequestDuration.WithLabelValues(serviceName, handler, method, status).Observe(duration)
	}
}
