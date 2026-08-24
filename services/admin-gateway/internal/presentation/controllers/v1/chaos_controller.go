package v1

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	paymentv1 "github.com/myavarih/sre-hands-on/services/gen/go/payment/v1"
	"github.com/myavarih/sre-hands-on/services/admin-gateway/internal/presentation/controllers"
)

type ChaosController struct {
	paymentClient paymentv1.PaymentServiceClient
	logger        *slog.Logger
}

func NewChaosController(paymentClient paymentv1.PaymentServiceClient, logger *slog.Logger) *ChaosController {
	return &ChaosController{
		paymentClient: paymentClient,
		logger:        logger,
	}
}

type faultInjectionParams struct {
	LatencyMs int32   `json:"latency_ms"`
	ErrorRate float64 `json:"error_rate"`
	Enabled   bool    `json:"enabled"`
}

func (c *ChaosController) SetFaultInjection(ctx *gin.Context) {
	params, ok := controllers.Receive[faultInjectionParams](ctx)
	if !ok {
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := c.paymentClient.SetChaosConfig(reqCtx, &paymentv1.SetChaosConfigRequest{
		LatencyMs: params.LatencyMs,
		ErrorRate: params.ErrorRate,
		Enabled:   params.Enabled,
	})
	if err != nil {
		c.logger.Error("Failed to set chaos config", "error", err)
		controllers.RespondError(ctx, http.StatusInternalServerError, "Failed to apply chaos configuration", err.Error())
		return
	}

	c.logger.Warn("SRE Fault Injection Configured via Admin API",
		"enabled", params.Enabled,
		"latency_ms", params.LatencyMs,
		"error_rate", params.ErrorRate,
	)

	controllers.Respond(ctx, http.StatusOK, "Chaos configuration applied", gin.H{
		"success":        resp.GetSuccess(),
		"current_status": resp.GetCurrentStatus(),
	})
}
