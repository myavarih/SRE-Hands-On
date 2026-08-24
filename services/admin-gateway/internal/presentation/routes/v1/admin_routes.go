package v1

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/myavarih/sre-hands-on/services/admin-gateway/internal/presentation/controllers/v1"
)

func RegisterAdminRoutes(rg *gin.RouterGroup, adminCtrl *v1.AdminController, chaosCtrl *v1.ChaosController) {
	rg.GET("/stats", adminCtrl.GetStats)
	rg.POST("/products", adminCtrl.CreateProduct)
	rg.POST("/chaos/fault-injection", chaosCtrl.SetFaultInjection)
}
