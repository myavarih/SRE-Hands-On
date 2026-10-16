package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/myavarih/sre-hands-on/pkg/persistence"
	"github.com/myavarih/sre-hands-on/pkg/telemetry"
	"github.com/myavarih/sre-hands-on/services/payment-service/internal/application/usecase"
	"github.com/myavarih/sre-hands-on/services/payment-service/internal/presentation/controllers"
	paymentv1 "github.com/myavarih/sre-hands-on/services/gen/go/payment/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
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
	db, err := persistence.NewPostgresDBWithReplica(env.DatabaseDSN, env.DatabaseRODSN, logger)
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}
	uow := persistence.NewGormUnitOfWork(db)

	// 3. Application Layer (Use Cases)
	paymentUseCase := usecase.NewPaymentUseCase(uow, logger)

	// 4. Presentation Layer (gRPC Controller)
	paymentController := controllers.NewPaymentGrpcController(paymentUseCase)

	// 5. gRPC Server with Observability Interceptors
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			start := time.Now()
			resp, err := handler(ctx, req)
			duration := time.Since(start).Seconds()

			grpcService := "unknown"
			grpcMethod := "unknown"
			if strings.HasPrefix(info.FullMethod, "/") {
				grpcService = path.Dir(info.FullMethod[1:])
				grpcMethod = path.Base(info.FullMethod)
			}
			statusCode := status.Code(err).String()

			telemetry.GRPCRequestsTotal.WithLabelValues(grpcService, grpcMethod, statusCode).Inc()
			telemetry.GRPCHandlingDuration.WithLabelValues(grpcService, grpcMethod, statusCode).Observe(duration)

			return resp, err
		}),
	)
	paymentv1.RegisterPaymentServiceServer(grpcServer, paymentController)
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
	go func() {
		a.Logger.Info("Starting metrics server", "port", a.Env.MetricsPort)
		if err := a.MetricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.Logger.Error("Metrics server failure", "error", err)
		}
	}()

	lis, err := net.Listen("tcp", ":"+a.Env.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %s: %w", a.Env.GRPCPort, err)
	}

	a.Logger.Info("Starting gRPC server", "port", a.Env.GRPCPort)
	return a.GRPCServer.Serve(lis)
}

func (a *App) Stop(ctx context.Context) {
	a.Logger.Info("Shutting down payment-service...")
	a.GRPCServer.GracefulStop()
	_ = a.MetricsServer.Shutdown(ctx)
	if a.ShutdownTracer != nil {
		_ = a.ShutdownTracer(ctx)
	}
	a.Logger.Info("payment-service stopped successfully")
}
