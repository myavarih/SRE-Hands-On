package v1

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/controllers/v1"
)

func RegisterProductRoutes(rg *gin.RouterGroup, ctrl *v1.ProductController) {
	productGroup := rg.Group("/products")
	{
		productGroup.GET("", ctrl.ListProducts)
		productGroup.GET("/:id", ctrl.GetProduct)
	}
}
