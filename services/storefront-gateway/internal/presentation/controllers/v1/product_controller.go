package v1

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	orderv1 "github.com/myavarih/sre-hands-on/services/gen/go/order/v1"
	"github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/controllers"
)

type ProductController struct {
	orderClient orderv1.OrderServiceClient
	logger      *slog.Logger
}

func NewProductController(orderClient orderv1.OrderServiceClient, logger *slog.Logger) *ProductController {
	return &ProductController{
		orderClient: orderClient,
		logger:      logger,
	}
}

func (c *ProductController) ListProducts(ctx *gin.Context) {
	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := c.orderClient.ListProducts(reqCtx, &orderv1.ListProductsRequest{})
	if err != nil {
		c.logger.Error("ListProducts RPC failed", "error", err)
		controllers.RespondError(ctx, http.StatusInternalServerError, "Failed to fetch catalog", err.Error())
		return
	}

	products := make([]gin.H, len(resp.GetProducts()))
	for i, p := range resp.GetProducts() {
		products[i] = gin.H{
			"id":             p.GetId(),
			"name":           p.GetName(),
			"description":    p.GetDescription(),
			"price_cents":    p.GetPriceCents(),
			"stock_quantity": p.GetStockQuantity(),
		}
	}

	controllers.Respond(ctx, http.StatusOK, "Products retrieved", products)
}

func (c *ProductController) GetProduct(ctx *gin.Context) {
	productID := ctx.Param("id")
	if productID == "" {
		controllers.RespondError(ctx, http.StatusBadRequest, "Product ID is required", "")
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := c.orderClient.GetProduct(reqCtx, &orderv1.GetProductRequest{
		ProductId: productID,
	})
	if err != nil {
		controllers.RespondError(ctx, http.StatusNotFound, "Product not found", "")
		return
	}

	p := resp.GetProduct()
	controllers.Respond(ctx, http.StatusOK, "Product details retrieved", gin.H{
		"id":             p.GetId(),
		"name":           p.GetName(),
		"description":    p.GetDescription(),
		"price_cents":    p.GetPriceCents(),
		"stock_quantity": p.GetStockQuantity(),
	})
}
