package bootstrap

import "os"

type Environment struct {
	ServiceName        string
	DatabaseDSN        string
	DatabaseRODSN      string
	GRPCPort           string
	MetricsPort        string
	PaymentServiceAddr string
	OTelAddr           string
}

func LoadEnvironment() *Environment {
	return &Environment{
		ServiceName:        getEnv("SERVICE_NAME", "order-service"),
		DatabaseDSN:        getEnv("DATABASE_DSN", "host=localhost user=app_user password=app_password dbname=app_db port=5432 sslmode=disable"),
		DatabaseRODSN:      getEnv("DATABASE_RO_DSN", ""),
		GRPCPort:           getEnv("GRPC_PORT", "50052"),
		MetricsPort:        getEnv("METRICS_PORT", "9090"),
		PaymentServiceAddr: getEnv("PAYMENT_SERVICE_ADDR", "payment-service.sre-app.svc.cluster.local:50053"),
		OTelAddr:           getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector.sre-monitoring.svc.cluster.local:4317"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
