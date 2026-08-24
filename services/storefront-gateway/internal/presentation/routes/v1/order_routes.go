package v1

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/controllers/v1"
	"github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/middleware"
)

func RegisterOrderRoutes(rg *gin.RouterGroup, ctrl *v1.OrderController, authMiddleware *middleware.AuthMiddleware) {
	orderGroup := rg.Group("/orders")
	orderGroup.Use(authMiddleware.AuthRequired())
	{
		orderGroup.POST("", ctrl.CreateOrder)
		orderGroup.GET("", ctrl.ListOrders)
		orderGroup.GET("/:id", ctrl.GetOrder)
	}
}
