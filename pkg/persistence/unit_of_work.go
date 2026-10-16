package persistence

import (
	"context"

	"github.com/myavarih/sre-hands-on/pkg/ports"
	"gorm.io/gorm"
)

// GormUnitOfWork implements ports.UnitOfWork managing transactional boundaries.
// In Clean Architecture, this decouples the application use cases from raw database handles (*gorm.DB or *sql.Tx).
// See docs/learning/04-GO-BACKEND-DEEP-DIVE.md for details.
type GormUnitOfWork struct {
	db               *gorm.DB
	userRepo         ports.UserRepository
	productRepo      ports.ProductRepository
	orderRepo        ports.OrderRepository
	paymentRepo      ports.PaymentRepository
	notificationRepo ports.NotificationRepository
}

func NewGormUnitOfWork(db *gorm.DB) ports.UnitOfWork {
	return &GormUnitOfWork{
		db:               db,
		userRepo:         NewGormUserRepository(db),
		productRepo:      NewGormProductRepository(db),
		orderRepo:        NewGormOrderRepository(db),
		paymentRepo:      NewGormPaymentRepository(db),
		notificationRepo: NewGormNotificationRepository(db),
	}
}

func (u *GormUnitOfWork) Users() ports.UserRepository {
	return u.userRepo
}

func (u *GormUnitOfWork) Products() ports.ProductRepository {
	return u.productRepo
}

func (u *GormUnitOfWork) Orders() ports.OrderRepository {
	return u.orderRepo
}

func (u *GormUnitOfWork) Payments() ports.PaymentRepository {
	return u.paymentRepo
}

func (u *GormUnitOfWork) Notifications() ports.NotificationRepository {
	return u.notificationRepo
}

// WithTransaction runs the given closure inside an atomic database transaction.
// Mechanics:
// 1. GORM executes SQL `BEGIN`.
// 2. A scoped txFactory is created where all repositories share the same `tx` transaction handle.
// 3. If fn() returns nil, GORM executes `COMMIT`.
// 4. If fn() returns an error or panics, GORM automatically executes `ROLLBACK`.
func (u *GormUnitOfWork) WithTransaction(ctx context.Context, fn func(rf ports.RepositoryFactory) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txFactory := &GormUnitOfWork{
			db:               tx,
			userRepo:         NewGormUserRepository(tx),
			productRepo:      NewGormProductRepository(tx),
			orderRepo:        NewGormOrderRepository(tx),
			paymentRepo:      NewGormPaymentRepository(tx),
			notificationRepo: NewGormNotificationRepository(tx),
		}
		return fn(txFactory)
	})
}
