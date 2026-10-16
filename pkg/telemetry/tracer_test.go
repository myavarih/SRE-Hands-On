package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestEndpointFilterSampler(t *testing.T) {
	sampler := NewEndpointFilterSampler(sdktrace.AlwaysSample())

	tests := []struct {
		name       string
		spanName   string
		attributes []attribute.KeyValue
		expected   sdktrace.SamplingDecision
	}{
		{
			name:     "drop healthz endpoint in span name",
			spanName: "GET /healthz",
			expected: sdktrace.Drop,
		},
		{
			name:     "drop metrics endpoint in span name",
			spanName: "GET /metrics",
			expected: sdktrace.Drop,
		},
		{
			name:     "drop healthz in attribute",
			spanName: "HTTP GET",
			attributes: []attribute.KeyValue{
				attribute.String("http.target", "/healthz"),
			},
			expected: sdktrace.Drop,
		},
		{
			name:     "drop metrics in attribute",
			spanName: "HTTP GET",
			attributes: []attribute.KeyValue{
				attribute.String("http.path", "/metrics"),
			},
			expected: sdktrace.Drop,
		},
		{
			name:     "sample normal business endpoint",
			spanName: "POST /api/v1/store/orders",
			attributes: []attribute.KeyValue{
				attribute.String("http.target", "/api/v1/store/orders"),
			},
			expected: sdktrace.RecordAndSample,
		},
		{
			name:     "sample product catalog query",
			spanName: "GET /api/v1/store/products",
			expected: sdktrace.RecordAndSample,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := sampler.ShouldSample(sdktrace.SamplingParameters{
				ParentContext: context.Background(),
				TraceID:       trace.TraceID{1},
				Name:          tt.spanName,
				Attributes:    tt.attributes,
			})
			if res.Decision != tt.expected {
				t.Errorf("ShouldSample(%s) decision = %v; want %v", tt.spanName, res.Decision, tt.expected)
			}
		})
	}
}
