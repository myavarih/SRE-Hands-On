package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

func TracingMiddleware(serviceName string) gin.HandlerFunc {
	tracer := otel.GetTracerProvider().Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/healthz" || path == "/metrics" {
			c.Next()
			return
		}

		// 1. Extract context from incoming HTTP headers
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 2. Start root HTTP span
		spanName := c.Request.Method + " " + c.FullPath()
		if c.FullPath() == "" {
			spanName = c.Request.Method + " " + c.Request.URL.Path
		}
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)

		c.Next()

		// 3. Record response status on span
		span.SetAttributes(
			attribute.Int("http.status_code", c.Writer.Status()),
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.path", c.Request.URL.Path),
		)
	}
}
