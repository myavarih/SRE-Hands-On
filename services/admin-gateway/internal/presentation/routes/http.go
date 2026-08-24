package routes

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	ctrlsv1 "github.com/myavarih/sre-hands-on/services/admin-gateway/internal/presentation/controllers/v1"
	"github.com/myavarih/sre-hands-on/services/admin-gateway/internal/presentation/middleware"
	routesv1 "github.com/myavarih/sre-hands-on/services/admin-gateway/internal/presentation/routes/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RouterParams struct {
	ServiceName     string
	Logger          *slog.Logger
	AdminController *ctrlsv1.AdminController
	ChaosController *ctrlsv1.ChaosController
}

func SetupRouter(p RouterParams) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(middleware.RecoveryMiddleware(p.Logger))
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.LoggingMiddleware(p.Logger))
	router.Use(middleware.PrometheusMiddleware(p.ServiceName))

	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	adminV1 := router.Group("/api/v1/admin")
	{
		routesv1.RegisterAdminRoutes(adminV1, p.AdminController, p.ChaosController)
	}

	return router
}
