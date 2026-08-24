package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents an authenticated customer or admin in the system.
type User struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username     string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	Role         string    `gorm:"type:varchar(32);not null;default:'customer'" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// Product represents a purchasable item with inventory.
type Product struct {
	ID            string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name          string    `gorm:"type:varchar(255);not null" json:"name"`
	Description   string    `gorm:"type:text" json:"description"`
	PriceCents    int64     `gorm:"not null;check:price_cents >= 0" json:"price_cents"`
	StockQuantity int32     `gorm:"not null;check:stock_quantity >= 0" json:"stock_quantity"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// OrderStatus defines the lifecycle states of an order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusCompleted OrderStatus = "COMPLETED"
	OrderStatusFailed    OrderStatus = "FAILED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

// Order represents a customer purchase transaction.
type Order struct {
	ID              string      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          string      `gorm:"type:uuid;not null;index" json:"user_id"`
	TotalPriceCents int64       `gorm:"not null;check:total_price_cents >= 0" json:"total_price_cents"`
	Status          OrderStatus `gorm:"type:varchar(32);not null;default:'PENDING'" json:"status"`
	TransactionID   string      `gorm:"type:varchar(128)" json:"transaction_id,omitempty"`
	Items           []OrderItem `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	return nil
}

// OrderItem represents a line item in an order.
type OrderItem struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID        string    `gorm:"type:uuid;not null;index" json:"order_id"`
	ProductID      string    `gorm:"type:uuid;not null;index" json:"product_id"`
	ProductName    string    `gorm:"-" json:"product_name,omitempty"`
	Quantity       int32     `gorm:"not null;check:quantity > 0" json:"quantity"`
	UnitPriceCents int64     `gorm:"not null" json:"unit_price_cents"`
	CreatedAt      time.Time `json:"created_at"`
}

func (oi *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if oi.ID == "" {
		oi.ID = uuid.New().String()
	}
	return nil
}

// Payment represents the financial charge record for an order.
type Payment struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID        string    `gorm:"type:uuid;not null;index" json:"order_id"`
	UserID         string    `gorm:"type:uuid;not null;index" json:"user_id"`
	AmountCents    int64     `gorm:"not null" json:"amount_cents"`
	Status         string    `gorm:"type:varchar(32);not null" json:"status"` // SUCCESS or FAILED
	TransactionRef string    `gorm:"type:varchar(128);not null" json:"transaction_ref"`
	CreatedAt      time.Time `json:"created_at"`
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// Notification records sent user notifications.
type Notification struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"user_id"`
	Recipient string    `gorm:"type:varchar(255);not null" json:"recipient"`
	Channel   string    `gorm:"type:varchar(32);not null" json:"channel"` // EMAIL or SMS
	Message   string    `gorm:"type:text;not null" json:"message"`
	Status    string    `gorm:"type:varchar(32);not null" json:"status"` // SENT or FAILED
	CreatedAt time.Time `json:"created_at"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	return nil
}
