package dto

import "time"

type ProcessPaymentInput struct {
	OrderID       string
	UserID        string
	AmountCents   int64
	PaymentMethod string
}

type PaymentOutput struct {
	ID             string
	OrderID        string
	AmountCents    int64
	Status         string
	TransactionRef string
	ProcessedAt    time.Time
}

type SendNotificationInput struct {
	UserID    string
	Recipient string
	Channel   string
	Message   string
}

type NotificationOutput struct {
	ID     string
	Status string
}

type SetChaosConfigInput struct {
	LatencyMs int32
	ErrorRate float64
	Enabled   bool
}

type ChaosConfigOutput struct {
	Success       bool
	CurrentStatus string
}
