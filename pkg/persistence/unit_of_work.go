package persistence

import (
	"context"

	"github.com/myavarih/sre-hands-on/pkg/ports"
	"gorm.io/gorm"
)

// GormUnitOfWork implements ports.UnitOfWork managing transactional scopes.
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

// WithTransaction runs the given function inside a database transaction.
// If the function returns an error, the transaction is automatically rolled back.
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
