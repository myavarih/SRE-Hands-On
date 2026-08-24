package ports

import (
	"context"

	"github.com/myavarih/sre-hands-on/pkg/entities"
)

// UserRepository handles user persistence operations.
type UserRepository interface {
	Create(ctx context.Context, user *entities.User) error
	FindByEmail(ctx context.Context, email string) (*entities.User, error)
	FindByUsername(ctx context.Context, username string) (*entities.User, error)
	FindByID(ctx context.Context, id string) (*entities.User, error)
	Count(ctx context.Context) (int64, error)
}

// ProductRepository handles product catalog and inventory persistence.
type ProductRepository interface {
	FindAll(ctx context.Context) ([]entities.Product, error)
	FindByID(ctx context.Context, id string) (*entities.Product, error)
	Create(ctx context.Context, product *entities.Product) error
	DeductStock(ctx context.Context, id string, quantity int32) error
	AddStock(ctx context.Context, id string, quantity int32) error
}

// OrderRepository handles order lifecycle persistence.
type OrderRepository interface {
	Create(ctx context.Context, order *entities.Order) error
	FindByID(ctx context.Context, id string) (*entities.Order, error)
	FindByUserID(ctx context.Context, userID string) ([]entities.Order, error)
	UpdateStatus(ctx context.Context, id string, status entities.OrderStatus, transactionID string) error
	Count(ctx context.Context) (int64, error)
	TotalRevenueCents(ctx context.Context) (int64, error)
}

// PaymentRepository handles payment transaction records.
type PaymentRepository interface {
	Create(ctx context.Context, payment *entities.Payment) error
	FindByID(ctx context.Context, id string) (*entities.Payment, error)
	FindByOrderID(ctx context.Context, orderID string) (*entities.Payment, error)
}

// NotificationRepository handles notification logs.
type NotificationRepository interface {
	Create(ctx context.Context, notif *entities.Notification) error
	FindByUserID(ctx context.Context, userID string) ([]entities.Notification, error)
}

// RepositoryFactory provides access to all repositories.
type RepositoryFactory interface {
	Users() UserRepository
	Products() ProductRepository
	Orders() OrderRepository
	Payments() PaymentRepository
	Notifications() NotificationRepository
}

// UnitOfWork coordinates atomic database transactions across multiple repositories.
type UnitOfWork interface {
	RepositoryFactory
	WithTransaction(ctx context.Context, fn func(rf RepositoryFactory) error) error
}
