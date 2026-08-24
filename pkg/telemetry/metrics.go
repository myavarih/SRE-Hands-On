package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal tracks total HTTP requests partitioned by path, method, and status code.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed by gateway handlers",
		},
		[]string{"service", "handler", "method", "status"},
	)

	// HTTPRequestDuration tracks latency of HTTP requests in seconds.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency of HTTP requests in seconds",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"service", "handler", "method", "status"},
	)

	// GRPCRequestsTotal tracks total gRPC server calls.
	GRPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_handled_total",
			Help: "Total number of completed gRPC requests",
		},
		[]string{"grpc_service", "grpc_method", "grpc_code"},
	)

	// GRPCHandlingDuration tracks latency of gRPC server calls in seconds.
	GRPCHandlingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_handling_seconds",
			Help:    "Latency of gRPC requests in seconds",
			Buckets: []float64{0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"grpc_service", "grpc_method", "grpc_code"},
	)

	// RateLimitRejections tracks rate limit drops at the gateway.
	RateLimitRejections = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_rate_limit_rejections_total",
			Help: "Total number of requests rejected by rate limiting",
		},
		[]string{"service", "route"},
	)
)
