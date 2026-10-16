package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal tracks total HTTP requests partitioned by service, handler, method, and status code.
	// Used to compute Throughput (Rate) and Error Ratio in the RED Method.
	// PromQL Throughput: sum(rate(http_requests_total[1m])) by (service, handler)
	// PromQL Error Rate:  sum(rate(http_requests_total{status=~"5.."}[1m])) / sum(rate(http_requests_total[1m]))
	// Note: See docs/learning/05-OBSERVABILITY-METRICS.md and Flaw F-03 for label alignment.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed by gateway handlers",
		},
		[]string{"service", "handler", "method", "status"},
	)

	// HTTPRequestDuration tracks latency of HTTP requests in seconds across cumulative buckets.
	// Used to compute Latency Percentiles (P50, P90, P99) in the RED Method.
	// PromQL P99: histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[2m])) by (le, service))
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency of HTTP requests in seconds",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"service", "handler", "method", "status"},
	)

	// GRPCRequestsTotal tracks total completed gRPC server calls.
	// Provides internal service-to-service east-west traffic observability.
	GRPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_handled_total",
			Help: "Total number of completed gRPC requests",
		},
		[]string{"grpc_service", "grpc_method", "grpc_code"},
	)

	// GRPCHandlingDuration tracks latency of internal gRPC server calls in seconds.
	GRPCHandlingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_handling_seconds",
			Help:    "Latency of gRPC requests in seconds",
			Buckets: []float64{0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"grpc_service", "grpc_method", "grpc_code"},
	)

	// RateLimitRejections tracks rate limit drops at the gateway (Token Bucket exhaustion).
	// Serves as a key Saturation and Abuse indicator for API Gateways.
	RateLimitRejections = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_rate_limit_rejections_total",
			Help: "Total number of requests rejected by rate limiting",
		},
		[]string{"service", "route"},
	)
)
