package v1

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	authv1 "github.com/myavarih/sre-hands-on/services/gen/go/auth/v1"
	orderv1 "github.com/myavarih/sre-hands-on/services/gen/go/order/v1"
	paymentv1 "github.com/myavarih/sre-hands-on/services/gen/go/payment/v1"
	"github.com/myavarih/sre-hands-on/services/admin-gateway/internal/presentation/controllers"
)

type AdminController struct {
	authClient    authv1.AuthServiceClient
	orderClient   orderv1.OrderServiceClient
	paymentClient paymentv1.PaymentServiceClient
	logger        *slog.Logger
}

func NewAdminController(
	authClient authv1.AuthServiceClient,
	orderClient orderv1.OrderServiceClient,
	paymentClient paymentv1.PaymentServiceClient,
	logger *slog.Logger,
) *AdminController {
	return &AdminController{
		authClient:    authClient,
		orderClient:   orderClient,
		paymentClient: paymentClient,
		logger:        logger,
	}
}

func (c *AdminController) GetStats(ctx *gin.Context) {
	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	prodResp, err := c.orderClient.ListProducts(reqCtx, &orderv1.ListProductsRequest{})
	if err != nil {
		c.logger.Error("Failed to fetch products for admin stats", "error", err)
		controllers.RespondError(ctx, http.StatusInternalServerError, "Failed to retrieve statistics", err.Error())
		return
	}

	totalProducts := len(prodResp.GetProducts())
	var totalStock int32
	for _, p := range prodResp.GetProducts() {
		totalStock += p.GetStockQuantity()
	}

	controllers.Respond(ctx, http.StatusOK, "System statistics retrieved", gin.H{
		"catalog_products_count": totalProducts,
		"total_stock_units":      totalStock,
		"system_health":          "OPERATIONAL",
		"timestamp":              time.Now().UTC().Format(time.RFC3339),
	})
}

type createProductParams struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	PriceCents   int64  `json:"price_cents" binding:"required,min=0"`
	InitialStock int32  `json:"initial_stock" binding:"required,min=0"`
}

func (c *AdminController) CreateProduct(ctx *gin.Context) {
	params, ok := controllers.Receive[createProductParams](ctx)
	if !ok {
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := c.orderClient.CreateProduct(reqCtx, &orderv1.CreateProductRequest{
		Name:         params.Name,
		Description:  params.Description,
		PriceCents:   params.PriceCents,
		InitialStock: params.InitialStock,
	})
	if err != nil {
		c.logger.Error("CreateProduct RPC failed", "error", err)
		controllers.RespondError(ctx, http.StatusBadRequest, "Failed to create product", err.Error())
		return
	}

	p := resp.GetProduct()
	controllers.Respond(ctx, http.StatusCreated, "Product created in catalog", gin.H{
		"id":             p.GetId(),
		"name":           p.GetName(),
		"description":    p.GetDescription(),
		"price_cents":    p.GetPriceCents(),
		"stock_quantity": p.GetStockQuantity(),
	})
}
