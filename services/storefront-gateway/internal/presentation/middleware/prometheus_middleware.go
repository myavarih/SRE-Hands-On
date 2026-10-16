package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/myavarih/sre-hands-on/pkg/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
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

		// Record Prometheus metric with OpenTelemetry trace_id exemplar (Flaw F-08 fix)
		observer := telemetry.HTTPRequestDuration.WithLabelValues(serviceName, handler, method, status)
		span := trace.SpanFromContext(c.Request.Context())
		if span.SpanContext().IsValid() {
			if exemplarObserver, ok := observer.(prometheus.ExemplarObserver); ok {
				exemplarObserver.ObserveWithExemplar(duration, prometheus.Labels{
					"trace_id": span.SpanContext().TraceID().String(),
				})
				return
			}
		}
		observer.Observe(duration)
	}
}
