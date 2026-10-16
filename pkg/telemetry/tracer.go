package telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// InitTracer initializes the OpenTelemetry tracer provider with an OTLP gRPC exporter.
// In SRE architecture, tracing captures causal relationships across network hops.
// See docs/learning/06-OBSERVABILITY-TRACING.md for the complete tracing deep dive.
func InitTracer(ctx context.Context, serviceName, collectorAddr string) (func(context.Context) error, error) {
	// Fallback to in-cluster OpenTelemetry Collector Kubernetes DNS service
	if collectorAddr == "" {
		collectorAddr = "otel-collector.sre-monitoring.svc.cluster.local:4317"
	}

	// 1. Resource Attributes: Identifies this service instance in Grafana Tempo / Jaeger.
	// Semantic conventions ensure standard querying across multi-language microservices.
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create otel resource: %w", err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// 2. OTLP Exporter: Transmits spans over gRPC to the OpenTelemetry Collector on port 4317.
	exporter, err := otlptracegrpc.New(ctxTimeout,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(collectorAddr),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		// Non-blocking fallback for local unit tests when no collector is reachable
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		return tp.Shutdown, nil
	}

	// 3. TracerProvider & BatchSpanProcessor:
	// Use EndpointFilterSampler to prevent noise: /healthz and /metrics spans are dropped.
	// AlwaysSample() captures 100% of all user and business traces.
	// WithBatchTimeout(500ms) queues spans in memory and flushes asynchronously,
	// ensuring telemetry never introduces latency to the critical user request path.
	sampler := NewEndpointFilterSampler(sdktrace.AlwaysSample())
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(500*time.Millisecond)),
		sdktrace.WithResource(res),
	)

	// 4. Global Registration & W3C Propagators:
	// Registers W3C TraceContext (traceparent header) and Baggage as global propagators.
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// EndpointFilterSampler drops traces for high-frequency operational endpoints (/healthz, /metrics)
// while delegating business requests to the underlying sampler.
type EndpointFilterSampler struct {
	delegate sdktrace.Sampler
}

// NewEndpointFilterSampler creates a sampler that filters out health check and metric endpoints.
func NewEndpointFilterSampler(delegate sdktrace.Sampler) sdktrace.Sampler {
	return &EndpointFilterSampler{delegate: delegate}
}

func (s *EndpointFilterSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	// 1. Check span name (e.g. "GET /healthz", "GET /metrics")
	nameLower := strings.ToLower(p.Name)
	if strings.Contains(nameLower, "/healthz") || strings.Contains(nameLower, "/metrics") {
		return sdktrace.SamplingResult{Decision: sdktrace.Drop}
	}

	// 2. Check semantic attributes (http.target, http.path, url.path)
	for _, attr := range p.Attributes {
		key := string(attr.Key)
		if key == "http.target" || key == "http.path" || key == "url.path" {
			val := strings.ToLower(attr.Value.AsString())
			if strings.Contains(val, "/healthz") || strings.Contains(val, "/metrics") {
				return sdktrace.SamplingResult{Decision: sdktrace.Drop}
			}
		}
	}

	return s.delegate.ShouldSample(p)
}

func (s *EndpointFilterSampler) Description() string {
	return fmt.Sprintf("EndpointFilterSampler{%s}", s.delegate.Description())
}
