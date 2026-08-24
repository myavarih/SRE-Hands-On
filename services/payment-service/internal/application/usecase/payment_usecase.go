package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/myavarih/sre-hands-on/pkg/entities"
	"github.com/myavarih/sre-hands-on/pkg/exceptions"
	"github.com/myavarih/sre-hands-on/pkg/ports"
	"github.com/myavarih/sre-hands-on/services/payment-service/internal/application/dto"
)

type PaymentUseCase interface {
	ProcessPayment(ctx context.Context, input dto.ProcessPaymentInput) (*dto.PaymentOutput, error)
	GetPaymentByID(ctx context.Context, paymentID string) (*dto.PaymentOutput, error)
	SendNotification(ctx context.Context, input dto.SendNotificationInput) (*dto.NotificationOutput, error)
	SetChaosConfig(ctx context.Context, input dto.SetChaosConfigInput) (*dto.ChaosConfigOutput, error)
}

type paymentUseCase struct {
	uow    ports.UnitOfWork
	logger *slog.Logger

	// SRE Chaos Injection State
	chaosMu       sync.RWMutex
	chaosEnabled  bool
	chaosLatency  time.Duration
	chaosErrorPct float64
}

func NewPaymentUseCase(uow ports.UnitOfWork, logger *slog.Logger) PaymentUseCase {
	return &paymentUseCase{
		uow:    uow,
		logger: logger,
	}
}

func (uc *paymentUseCase) ProcessPayment(ctx context.Context, input dto.ProcessPaymentInput) (*dto.PaymentOutput, error) {
	if input.OrderID == "" || input.UserID == "" || input.AmountCents <= 0 {
		return nil, exceptions.NewValidationError("order_id, user_id, and positive amount_cents are required")
	}

	// 1. Evaluate SRE Chaos Injection
	uc.chaosMu.RLock()
	enabled := uc.chaosEnabled
	latency := uc.chaosLatency
	errorPct := uc.chaosErrorPct
	uc.chaosMu.RUnlock()

	if enabled {
		if latency > 0 {
			uc.logger.Warn("SRE Chaos: Injecting artificial latency", "duration", latency)
			time.Sleep(latency)
		}
		if errorPct > 0 && rand.Float64() < errorPct {
			uc.logger.Warn("SRE Chaos: Injecting artificial payment failure", "error_rate", errorPct)
			return nil, exceptions.NewInternalError("payment gateway simulation error (chaos injected)", nil)
		}
	}

	// 2. Generate Transaction Reference
	txRef := fmt.Sprintf("txn_%s", uuid.New().String()[:12])
	payment := &entities.Payment{
		OrderID:        input.OrderID,
		UserID:         input.UserID,
		AmountCents:    input.AmountCents,
		Status:         "SUCCESS",
		TransactionRef: txRef,
	}

	if err := uc.uow.Payments().Create(ctx, payment); err != nil {
		uc.logger.Error("Failed to persist payment record", "error", err)
		return nil, err
	}

	// 3. Asynchronously record notification dispatch
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		notif := &entities.Notification{
			UserID:    input.UserID,
			Recipient: fmt.Sprintf("user-%s@example.com", input.UserID[:8]),
			Channel:   "EMAIL",
			Message:   fmt.Sprintf("Payment of $%.2f confirmed for Order %s", float64(input.AmountCents)/100.0, input.OrderID),
			Status:    "SENT",
		}
		_ = uc.uow.Notifications().Create(bgCtx, notif)
	}()

	uc.logger.Info("Payment processed successfully", "payment_id", payment.ID, "order_id", input.OrderID, "amount_cents", input.AmountCents)

	return &dto.PaymentOutput{
		ID:             payment.ID,
		OrderID:        payment.OrderID,
		AmountCents:    payment.AmountCents,
		Status:         payment.Status,
		TransactionRef: payment.TransactionRef,
		ProcessedAt:    payment.CreatedAt,
	}, nil
}

func (uc *paymentUseCase) GetPaymentByID(ctx context.Context, paymentID string) (*dto.PaymentOutput, error) {
	if paymentID == "" {
		return nil, exceptions.NewValidationError("payment_id is required")
	}

	payment, err := uc.uow.Payments().FindByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	return &dto.PaymentOutput{
		ID:             payment.ID,
		OrderID:        payment.OrderID,
		AmountCents:    payment.AmountCents,
		Status:         payment.Status,
		TransactionRef: payment.TransactionRef,
		ProcessedAt:    payment.CreatedAt,
	}, nil
}

func (uc *paymentUseCase) SendNotification(ctx context.Context, input dto.SendNotificationInput) (*dto.NotificationOutput, error) {
	if input.UserID == "" || input.Recipient == "" || input.Message == "" {
		return nil, exceptions.NewValidationError("user_id, recipient, and message are required")
	}

	notif := &entities.Notification{
		UserID:    input.UserID,
		Recipient: input.Recipient,
		Channel:   input.Channel,
		Message:   input.Message,
		Status:    "SENT",
	}

	if err := uc.uow.Notifications().Create(ctx, notif); err != nil {
		return nil, err
	}

	return &dto.NotificationOutput{
		ID:     notif.ID,
		Status: notif.Status,
	}, nil
}

func (uc *paymentUseCase) SetChaosConfig(ctx context.Context, input dto.SetChaosConfigInput) (*dto.ChaosConfigOutput, error) {
	uc.chaosMu.Lock()
	uc.chaosEnabled = input.Enabled
	uc.chaosLatency = time.Duration(input.LatencyMs) * time.Millisecond
	uc.chaosErrorPct = input.ErrorRate
	uc.chaosMu.Unlock()

	statusMsg := fmt.Sprintf("Chaos config updated: enabled=%v, latency=%dms, error_rate=%.2f",
		input.Enabled, input.LatencyMs, input.ErrorRate)
	uc.logger.Info("SRE Chaos configuration changed", "enabled", input.Enabled, "latency_ms", input.LatencyMs, "error_rate", input.ErrorRate)

	return &dto.ChaosConfigOutput{
		Success:       true,
		CurrentStatus: statusMsg,
	}, nil
}
