package dto

import "time"

type ProductDTO struct {
	ID            string
	Name          string
	Description   string
	PriceCents    int64
	StockQuantity int32
}

type CreateProductInput struct {
	Name         string
	Description  string
	PriceCents   int64
	InitialStock int32
}

type OrderItemInput struct {
	ProductID string
	Quantity  int32
}

type CreateOrderInput struct {
	UserID        string
	Items         []OrderItemInput
	PaymentMethod string
}

type OrderItemDTO struct {
	ProductID      string
	ProductName    string
	Quantity       int32
	UnitPriceCents int64
}

type OrderDTO struct {
	ID              string
	UserID          string
	TotalPriceCents int64
	Status          string
	TransactionID   string
	CreatedAt       time.Time
	Items           []OrderItemDTO
}
