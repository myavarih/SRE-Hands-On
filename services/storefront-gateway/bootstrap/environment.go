package bootstrap

import "os"

type Environment struct {
	ServiceName      string
	HTTPPort         string
	MetricsPort      string
	AuthServiceAddr  string
	OrderServiceAddr string
	JWTSecret        string
	OTelAddr         string
}

func LoadEnvironment() *Environment {
	return &Environment{
		ServiceName:      getEnv("SERVICE_NAME", "storefront-gateway"),
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		MetricsPort:      getEnv("METRICS_PORT", "9090"),
		AuthServiceAddr:  getEnv("AUTH_SERVICE_ADDR", "auth-service.sre-app.svc.cluster.local:50051"),
		OrderServiceAddr: getEnv("ORDER_SERVICE_ADDR", "order-service.sre-app.svc.cluster.local:50052"),
		JWTSecret:        getEnv("JWT_SECRET", "sre-super-secret-jwt-key-change-in-prod"),
		OTelAddr:         getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector.sre-monitoring.svc.cluster.local:4317"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
