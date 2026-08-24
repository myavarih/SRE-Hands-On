package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/myavarih/sre-hands-on/pkg/telemetry"
	authv1 "github.com/myavarih/sre-hands-on/services/gen/go/auth/v1"
	orderv1 "github.com/myavarih/sre-hands-on/services/gen/go/order/v1"
	paymentv1 "github.com/myavarih/sre-hands-on/services/gen/go/payment/v1"
	v1 "github.com/myavarih/sre-hands-on/services/admin-gateway/internal/presentation/controllers/v1"
	"github.com/myavarih/sre-hands-on/services/admin-gateway/internal/presentation/routes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type App struct {
	Env            *Environment
	Logger         *slog.Logger
	HTTPServer     *http.Server
	AuthConn       *grpc.ClientConn
	OrderConn      *grpc.ClientConn
	PaymentConn    *grpc.ClientConn
	ShutdownTracer func(context.Context) error
}

func InitializeApp(ctx context.Context) (*App, error) {
	env := LoadEnvironment()
	logger := telemetry.NewLogger(env.ServiceName)

	// 1. Initialize Telemetry
	shutdownTracer, err := telemetry.InitTracer(ctx, env.ServiceName, env.OTelAddr)
	if err != nil {
		logger.Warn("OTel tracer initialization skipped", "error", err)
	}

	// 2. Outbound gRPC Connections
	authConn, err := grpc.NewClient(env.AuthServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial auth-service: %w", err)
	}
	authClient := authv1.NewAuthServiceClient(authConn)

	orderConn, err := grpc.NewClient(env.OrderServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial order-service: %w", err)
	}
	orderClient := orderv1.NewOrderServiceClient(orderConn)

	paymentConn, err := grpc.NewClient(env.PaymentServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial payment-service: %w", err)
	}
	paymentClient := paymentv1.NewPaymentServiceClient(paymentConn)

	// 3. Controllers
	adminController := v1.NewAdminController(authClient, orderClient, paymentClient, logger)
	chaosController := v1.NewChaosController(paymentClient, logger)

	// 4. Setup Gin Router
	engine := routes.SetupRouter(routes.RouterParams{
		ServiceName:     env.ServiceName,
		Logger:          logger,
		AdminController: adminController,
		ChaosController: chaosController,
	})

	httpServer := &http.Server{
		Addr:    ":" + env.HTTPPort,
		Handler: engine,
	}

	return &App{
		Env:            env,
		Logger:         logger,
		HTTPServer:     httpServer,
		AuthConn:       authConn,
		OrderConn:      orderConn,
		PaymentConn:    paymentConn,
		ShutdownTracer: shutdownTracer,
	}, nil
}

func (a *App) Run() error {
	a.Logger.Info("Starting admin-gateway HTTP server", "port", a.Env.HTTPPort)
	return a.HTTPServer.ListenAndServe()
}

func (a *App) Stop(ctx context.Context) {
	a.Logger.Info("Shutting down admin-gateway...")
	_ = a.HTTPServer.Shutdown(ctx)
	if a.AuthConn != nil {
		_ = a.AuthConn.Close()
	}
	if a.OrderConn != nil {
		_ = a.OrderConn.Close()
	}
	if a.PaymentConn != nil {
		_ = a.PaymentConn.Close()
	}
	if a.ShutdownTracer != nil {
		_ = a.ShutdownTracer(ctx)
	}
	a.Logger.Info("admin-gateway stopped successfully")
}
