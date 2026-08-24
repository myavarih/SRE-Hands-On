package v1

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/controllers/v1"
)

func RegisterAuthRoutes(rg *gin.RouterGroup, ctrl *v1.AuthController) {
	authGroup := rg.Group("/auth")
	{
		authGroup.POST("/register", ctrl.Register)
		authGroup.POST("/login", ctrl.Login)
	}
}
