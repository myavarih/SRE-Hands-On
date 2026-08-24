package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/myavarih/sre-hands-on/pkg/persistence"
	"github.com/myavarih/sre-hands-on/pkg/telemetry"
	"github.com/myavarih/sre-hands-on/services/auth-service/internal/application/usecase"
	"github.com/myavarih/sre-hands-on/services/auth-service/internal/presentation/controllers"
	authv1 "github.com/myavarih/sre-hands-on/services/gen/go/auth/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type App struct {
	Env            *Environment
	Logger         *slog.Logger
	GRPCServer     *grpc.Server
	MetricsServer  *http.Server
	ShutdownTracer func(context.Context) error
}

func InitializeApp(ctx context.Context) (*App, error) {
	env := LoadEnvironment()
	logger := telemetry.NewLogger(env.ServiceName)

	// 1. Telemetry
	shutdownTracer, err := telemetry.InitTracer(ctx, env.ServiceName, env.OTelAddr)
	if err != nil {
		logger.Warn("OTel tracer initialization skipped", "error", err)
	}

	// 2. Persistence Layer
	db, err := persistence.NewPostgresDB(env.DatabaseDSN, logger)
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}
	uow := persistence.NewGormUnitOfWork(db)

	// 3. Application Layer (Use Cases)
	authUseCase := usecase.NewAuthUseCase(uow, env.JWTSecret, logger)

	// 4. Presentation Layer (gRPC Controller)
	authController := controllers.NewAuthGrpcController(authUseCase)

	// 5. gRPC Server with Observability Interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			start := time.Now()
			resp, err := handler(ctx, req)
			duration := time.Since(start).Seconds()

			code := "OK"
			if err != nil {
				code = "ERROR"
			}
			telemetry.GRPCRequestsTotal.WithLabelValues(env.ServiceName, info.FullMethod, code).Inc()
			telemetry.GRPCHandlingDuration.WithLabelValues(env.ServiceName, info.FullMethod, code).Observe(duration)

			return resp, err
		}),
	)
	authv1.RegisterAuthServiceServer(grpcServer, authController)
	reflection.Register(grpcServer)

	// 6. Metrics HTTP Server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	metricsServer := &http.Server{
		Addr:    ":" + env.MetricsPort,
		Handler: metricsMux,
	}

	return &App{
		Env:            env,
		Logger:         logger,
		GRPCServer:     grpcServer,
		MetricsServer:  metricsServer,
		ShutdownTracer: shutdownTracer,
	}, nil
}

func (a *App) Run() error {
	// Start metrics server
	go func() {
		a.Logger.Info("Starting metrics server", "port", a.Env.MetricsPort)
		if err := a.MetricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.Logger.Error("Metrics server failure", "error", err)
		}
	}()

	// Start gRPC server
	lis, err := net.Listen("tcp", ":"+a.Env.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %s: %w", a.Env.GRPCPort, err)
	}

	a.Logger.Info("Starting gRPC server", "port", a.Env.GRPCPort)
	return a.GRPCServer.Serve(lis)
}

func (a *App) Stop(ctx context.Context) {
	a.Logger.Info("Shutting down auth-service...")
	a.GRPCServer.GracefulStop()
	_ = a.MetricsServer.Shutdown(ctx)
	if a.ShutdownTracer != nil {
		_ = a.ShutdownTracer(ctx)
	}
	a.Logger.Info("auth-service stopped successfully")
}
