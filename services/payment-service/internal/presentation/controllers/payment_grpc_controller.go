package controllers

import (
	"context"
	"time"

	"github.com/myavarih/sre-hands-on/pkg/exceptions"
	paymentv1 "github.com/myavarih/sre-hands-on/services/gen/go/payment/v1"
	"github.com/myavarih/sre-hands-on/services/payment-service/internal/application/dto"
	"github.com/myavarih/sre-hands-on/services/payment-service/internal/application/usecase"
)

// PaymentGrpcController is a thin transport adapter mapping Protobuf to Application DTOs.
type PaymentGrpcController struct {
	paymentv1.UnimplementedPaymentServiceServer
	paymentUC usecase.PaymentUseCase
}

func NewPaymentGrpcController(paymentUC usecase.PaymentUseCase) *PaymentGrpcController {
	return &PaymentGrpcController{paymentUC: paymentUC}
}

func (c *PaymentGrpcController) ProcessPayment(ctx context.Context, req *paymentv1.ProcessPaymentRequest) (*paymentv1.ProcessPaymentResponse, error) {
	out, err := c.paymentUC.ProcessPayment(ctx, dto.ProcessPaymentInput{
		OrderID:       req.GetOrderId(),
		UserID:        req.GetUserId(),
		AmountCents:   req.GetAmountCents(),
		PaymentMethod: req.GetPaymentMethod(),
	})
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	return &paymentv1.ProcessPaymentResponse{
		PaymentId:      out.ID,
		OrderId:        out.OrderID,
		TransactionRef: out.TransactionRef,
		Status:         out.Status,
		Message:        "Payment captured successfully",
		ProcessedAt:    out.ProcessedAt.Format(time.RFC3339),
	}, nil
}

func (c *PaymentGrpcController) GetPaymentStatus(ctx context.Context, req *paymentv1.GetPaymentStatusRequest) (*paymentv1.GetPaymentStatusResponse, error) {
	out, err := c.paymentUC.GetPaymentByID(ctx, req.GetPaymentId())
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	return &paymentv1.GetPaymentStatusResponse{
		PaymentId:      out.ID,
		OrderId:        out.OrderID,
		AmountCents:    out.AmountCents,
		Status:         out.Status,
		TransactionRef: out.TransactionRef,
	}, nil
}

func (c *PaymentGrpcController) SendNotification(ctx context.Context, req *paymentv1.SendNotificationRequest) (*paymentv1.SendNotificationResponse, error) {
	out, err := c.paymentUC.SendNotification(ctx, dto.SendNotificationInput{
		UserID:    req.GetUserId(),
		Recipient: req.GetRecipient(),
		Channel:   req.GetChannel(),
		Message:   req.GetMessage(),
	})
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	return &paymentv1.SendNotificationResponse{
		NotificationId: out.ID,
		Status:         out.Status,
	}, nil
}

func (c *PaymentGrpcController) SetChaosConfig(ctx context.Context, req *paymentv1.SetChaosConfigRequest) (*paymentv1.SetChaosConfigResponse, error) {
	out, err := c.paymentUC.SetChaosConfig(ctx, dto.SetChaosConfigInput{
		LatencyMs: req.GetLatencyMs(),
		ErrorRate: req.GetErrorRate(),
		Enabled:   req.GetEnabled(),
	})
	if err != nil {
		return nil, exceptions.ToGRPCError(err)
	}

	return &paymentv1.SetChaosConfigResponse{
		Success:       out.Success,
		CurrentStatus: out.CurrentStatus,
	}, nil
}
