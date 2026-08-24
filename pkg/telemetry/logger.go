package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// TraceContextHandler is a custom slog.Handler that enriches log records with TraceID and SpanID.
type TraceContextHandler struct {
	slog.Handler
}

func (h *TraceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		r.AddAttrs(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *TraceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *TraceContextHandler) WithGroup(name string) slog.Handler {
	return &TraceContextHandler{Handler: h.Handler.WithGroup(name)}
}

// NewLogger creates a structured JSON logger with service name and trace context enrichment.
func NewLogger(serviceName string) *slog.Logger {
	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	handler := &TraceContextHandler{
		Handler: baseHandler.WithAttrs([]slog.Attr{
			slog.String("service", serviceName),
		}),
	}

	return slog.New(handler)
}
