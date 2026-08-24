package bootstrap

import (
	"os"
)

type Environment struct {
	ServiceName string
	DatabaseDSN string
	GRPCPort    string
	MetricsPort string
	JWTSecret   string
	OTelAddr    string
}

func LoadEnvironment() *Environment {
	serviceName := getEnv("SERVICE_NAME", "auth-service")
	databaseDSN := getEnv("DATABASE_DSN", "host=localhost user=app_user password=app_password dbname=app_db port=5432 sslmode=disable")
	grpcPort := getEnv("GRPC_PORT", "50051")
	metricsPort := getEnv("METRICS_PORT", "9090")
	jwtSecret := getEnv("JWT_SECRET", "sre-super-secret-jwt-key-change-in-prod")
	otelAddr := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector.sre-monitoring.svc.cluster.local:4317")

	return &Environment{
		ServiceName: serviceName,
		DatabaseDSN: databaseDSN,
		GRPCPort:    grpcPort,
		MetricsPort: metricsPort,
		JWTSecret:   jwtSecret,
		OTelAddr:    otelAddr,
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
