package routes

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	ctrlsv1 "github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/controllers/v1"
	"github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/middleware"
	routesv1 "github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/routes/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RouterParams struct {
	ServiceName       string
	Logger            *slog.Logger
	AuthController    *ctrlsv1.AuthController
	ProductController *ctrlsv1.ProductController
	OrderController   *ctrlsv1.OrderController
	AuthMiddleware    *middleware.AuthMiddleware
}

func SetupRouter(p RouterParams) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Global Middlewares
	router.Use(middleware.RecoveryMiddleware(p.Logger))
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.LoggingMiddleware(p.Logger))
	router.Use(middleware.PrometheusMiddleware(p.ServiceName))
	router.Use(middleware.RateLimitMiddleware(p.ServiceName, 100, 200))

	// Healthz & Metrics probes
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 Storefront Routes
	apiV1 := router.Group("/api/v1/store")
	{
		routesv1.RegisterAuthRoutes(apiV1, p.AuthController)
		routesv1.RegisterProductRoutes(apiV1, p.ProductController)
		routesv1.RegisterOrderRoutes(apiV1, p.OrderController, p.AuthMiddleware)
	}

	return router
}
