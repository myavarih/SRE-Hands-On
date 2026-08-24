package v1

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/myavarih/sre-hands-on/pkg/exceptions"
	orderv1 "github.com/myavarih/sre-hands-on/services/gen/go/order/v1"
	"github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/controllers"
)

type OrderController struct {
	orderClient orderv1.OrderServiceClient
	logger      *slog.Logger
}

func NewOrderController(orderClient orderv1.OrderServiceClient, logger *slog.Logger) *OrderController {
	return &OrderController{
		orderClient: orderClient,
		logger:      logger,
	}
}

type orderItemInput struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int32  `json:"quantity" binding:"required,min=1"`
}

type createOrderRequestParams struct {
	Items         []orderItemInput `json:"items" binding:"required,min=1"`
	PaymentMethod string           `json:"payment_method" binding:"required"`
}

func (c *OrderController) CreateOrder(ctx *gin.Context) {
	userID, err := controllers.GetUserID(ctx)
	if err != nil {
		controllers.RespondError(ctx, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	params, ok := controllers.Receive[createOrderRequestParams](ctx)
	if !ok {
		return
	}

	protoItems := make([]*orderv1.OrderItemInput, len(params.Items))
	for i, item := range params.Items {
		protoItems[i] = &orderv1.OrderItemInput{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 12*time.Second)
	defer cancel()

	resp, err := c.orderClient.CreateOrder(reqCtx, &orderv1.CreateOrderRequest{
		UserId:        userID,
		Items:         protoItems,
		PaymentMethod: params.PaymentMethod,
	})
	if err != nil {
		c.logger.Error("CreateOrder RPC failed", "error", err, "user_id", userID)
		dErr := exceptions.FromGRPCError(err)
		controllers.RespondError(ctx, http.StatusBadRequest, dErr.Error(), "")
		return
	}

	items := make([]gin.H, len(resp.GetItems()))
	for i, item := range resp.GetItems() {
		items[i] = gin.H{
			"product_id":       item.GetProductId(),
			"product_name":     item.GetProductName(),
			"quantity":         item.GetQuantity(),
			"unit_price_cents": item.GetUnitPriceCents(),
		}
	}

	controllers.Respond(ctx, http.StatusCreated, "Order created and payment processed", gin.H{
		"order_id":          resp.GetOrderId(),
		"user_id":           resp.GetUserId(),
		"total_price_cents": resp.GetTotalPriceCents(),
		"status":            resp.GetStatus(),
		"transaction_id":    resp.GetTransactionId(),
		"created_at":        resp.GetCreatedAt(),
		"items":             items,
	})
}

func (c *OrderController) GetOrder(ctx *gin.Context) {
	orderID := ctx.Param("id")
	if orderID == "" {
		controllers.RespondError(ctx, http.StatusBadRequest, "Order ID is required", "")
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := c.orderClient.GetOrder(reqCtx, &orderv1.GetOrderRequest{
		OrderId: orderID,
	})
	if err != nil {
		controllers.RespondError(ctx, http.StatusNotFound, "Order not found", "")
		return
	}

	items := make([]gin.H, len(resp.GetItems()))
	for i, item := range resp.GetItems() {
		items[i] = gin.H{
			"product_id":       item.GetProductId(),
			"quantity":         item.GetQuantity(),
			"unit_price_cents": item.GetUnitPriceCents(),
		}
	}

	controllers.Respond(ctx, http.StatusOK, "Order details retrieved", gin.H{
		"order_id":          resp.GetOrderId(),
		"user_id":           resp.GetUserId(),
		"total_price_cents": resp.GetTotalPriceCents(),
		"status":            resp.GetStatus(),
		"transaction_id":    resp.GetTransactionId(),
		"created_at":        resp.GetCreatedAt(),
		"items":             items,
	})
}

func (c *OrderController) ListOrders(ctx *gin.Context) {
	userID, err := controllers.GetUserID(ctx)
	if err != nil {
		controllers.RespondError(ctx, http.StatusUnauthorized, "Unauthorized", err.Error())
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := c.orderClient.ListUserOrders(reqCtx, &orderv1.ListUserOrdersRequest{
		UserId: userID,
	})
	if err != nil {
		controllers.RespondError(ctx, http.StatusInternalServerError, "Failed to retrieve order history", err.Error())
		return
	}

	orders := make([]gin.H, len(resp.GetOrders()))
	for i, o := range resp.GetOrders() {
		items := make([]gin.H, len(o.GetItems()))
		for j, item := range o.GetItems() {
			items[j] = gin.H{
				"product_id":       item.GetProductId(),
				"quantity":         item.GetQuantity(),
				"unit_price_cents": item.GetUnitPriceCents(),
			}
		}
		orders[i] = gin.H{
			"order_id":          o.GetOrderId(),
			"user_id":           o.GetUserId(),
			"total_price_cents": o.GetTotalPriceCents(),
			"status":            o.GetStatus(),
			"transaction_id":    o.GetTransactionId(),
			"created_at":        o.GetCreatedAt(),
			"items":             items,
		}
	}

	controllers.Respond(ctx, http.StatusOK, "User orders retrieved", orders)
}
