package persistence

import (
	"context"
	"errors"

	"github.com/myavarih/sre-hands-on/pkg/entities"
	"github.com/myavarih/sre-hands-on/pkg/exceptions"
	"github.com/myavarih/sre-hands-on/pkg/ports"
	"gorm.io/gorm"
)

// GormUserRepository implements ports.UserRepository using GORM.
type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) ports.UserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(ctx context.Context, user *entities.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return exceptions.NewAlreadyExistsError("user with this email or username already exists")
		}
		return exceptions.NewInternalError("failed to create user", err)
	}
	return nil
}

func (r *GormUserRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exceptions.NewNotFoundError("user not found")
		}
		return nil, exceptions.NewInternalError("failed to find user by email", err)
	}
	return &user, nil
}

func (r *GormUserRepository) FindByUsername(ctx context.Context, username string) (*entities.User, error) {
	var user entities.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exceptions.NewNotFoundError("user not found")
		}
		return nil, exceptions.NewInternalError("failed to find user by username", err)
	}
	return &user, nil
}

func (r *GormUserRepository) FindByID(ctx context.Context, id string) (*entities.User, error) {
	var user entities.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exceptions.NewNotFoundError("user not found")
		}
		return nil, exceptions.NewInternalError("failed to find user by id", err)
	}
	return &user, nil
}

func (r *GormUserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&entities.User{}).Count(&count).Error; err != nil {
		return 0, exceptions.NewInternalError("failed to count users", err)
	}
	return count, nil
}

// GormProductRepository implements ports.ProductRepository using GORM.
type GormProductRepository struct {
	db *gorm.DB
}

func NewGormProductRepository(db *gorm.DB) ports.ProductRepository {
	return &GormProductRepository{db: db}
}

func (r *GormProductRepository) FindAll(ctx context.Context) ([]entities.Product, error) {
	var products []entities.Product
	if err := r.db.WithContext(ctx).Order("created_at desc").Find(&products).Error; err != nil {
		return nil, exceptions.NewInternalError("failed to list products", err)
	}
	return products, nil
}

func (r *GormProductRepository) FindByID(ctx context.Context, id string) (*entities.Product, error) {
	var product entities.Product
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exceptions.NewNotFoundError("product not found")
		}
		return nil, exceptions.NewInternalError("failed to find product", err)
	}
	return &product, nil
}

func (r *GormProductRepository) Create(ctx context.Context, product *entities.Product) error {
	if err := r.db.WithContext(ctx).Create(product).Error; err != nil {
		return exceptions.NewInternalError("failed to create product", err)
	}
	return nil
}

func (r *GormProductRepository) DeductStock(ctx context.Context, id string, quantity int32) error {
	// Atomic row-level decrement enforcing stock >= quantity
	result := r.db.WithContext(ctx).
		Model(&entities.Product{}).
		Where("id = ? AND stock_quantity >= ?", id, quantity).
		Update("stock_quantity", gorm.Expr("stock_quantity - ?", quantity))

	if result.Error != nil {
		return exceptions.NewInternalError("failed to deduct stock", result.Error)
	}

	if result.RowsAffected == 0 {
		return exceptions.NewConflictError("insufficient product inventory or product not found")
	}

	return nil
}

func (r *GormProductRepository) AddStock(ctx context.Context, id string, quantity int32) error {
	result := r.db.WithContext(ctx).
		Model(&entities.Product{}).
		Where("id = ?", id).
		Update("stock_quantity", gorm.Expr("stock_quantity + ?", quantity))

	if result.Error != nil {
		return exceptions.NewInternalError("failed to add stock", result.Error)
	}
	return nil
}

// GormOrderRepository implements ports.OrderRepository using GORM.
type GormOrderRepository struct {
	db *gorm.DB
}

func NewGormOrderRepository(db *gorm.DB) ports.OrderRepository {
	return &GormOrderRepository{db: db}
}

func (r *GormOrderRepository) Create(ctx context.Context, order *entities.Order) error {
	if err := r.db.WithContext(ctx).Create(order).Error; err != nil {
		return exceptions.NewInternalError("failed to create order", err)
	}
	return nil
}

func (r *GormOrderRepository) FindByID(ctx context.Context, id string) (*entities.Order, error) {
	var order entities.Order
	if err := r.db.WithContext(ctx).Preload("Items").Where("id = ?", id).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exceptions.NewNotFoundError("order not found")
		}
		return nil, exceptions.NewInternalError("failed to find order", err)
	}
	return &order, nil
}

func (r *GormOrderRepository) FindByUserID(ctx context.Context, userID string) ([]entities.Order, error) {
	var orders []entities.Order
	if err := r.db.WithContext(ctx).Preload("Items").Where("user_id = ?", userID).Order("created_at desc").Find(&orders).Error; err != nil {
		return nil, exceptions.NewInternalError("failed to find user orders", err)
	}
	return orders, nil
}

func (r *GormOrderRepository) UpdateStatus(ctx context.Context, id string, status entities.OrderStatus, transactionID string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if transactionID != "" {
		updates["transaction_id"] = transactionID
	}

	result := r.db.WithContext(ctx).Model(&entities.Order{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return exceptions.NewInternalError("failed to update order status", result.Error)
	}
	return nil
}

func (r *GormOrderRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&entities.Order{}).Count(&count).Error; err != nil {
		return 0, exceptions.NewInternalError("failed to count orders", err)
	}
	return count, nil
}

func (r *GormOrderRepository) TotalRevenueCents(ctx context.Context) (int64, error) {
	var total *int64
	err := r.db.WithContext(ctx).
		Model(&entities.Order{}).
		Where("status = ?", entities.OrderStatusCompleted).
		Select("COALESCE(SUM(total_price_cents), 0)").
		Scan(&total).Error

	if err != nil {
		return 0, exceptions.NewInternalError("failed to calculate revenue", err)
	}

	if total == nil {
		return 0, nil
	}
	return *total, nil
}

// GormPaymentRepository implements ports.PaymentRepository using GORM.
type GormPaymentRepository struct {
	db *gorm.DB
}

func NewGormPaymentRepository(db *gorm.DB) ports.PaymentRepository {
	return &GormPaymentRepository{db: db}
}

func (r *GormPaymentRepository) Create(ctx context.Context, payment *entities.Payment) error {
	if err := r.db.WithContext(ctx).Create(payment).Error; err != nil {
		return exceptions.NewInternalError("failed to record payment", err)
	}
	return nil
}

func (r *GormPaymentRepository) FindByID(ctx context.Context, id string) (*entities.Payment, error) {
	var payment entities.Payment
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exceptions.NewNotFoundError("payment not found")
		}
		return nil, exceptions.NewInternalError("failed to find payment", err)
	}
	return &payment, nil
}

func (r *GormPaymentRepository) FindByOrderID(ctx context.Context, orderID string) (*entities.Payment, error) {
	var payment entities.Payment
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exceptions.NewNotFoundError("payment for order not found")
		}
		return nil, exceptions.NewInternalError("failed to find payment for order", err)
	}
	return &payment, nil
}

// GormNotificationRepository implements ports.NotificationRepository using GORM.
type GormNotificationRepository struct {
	db *gorm.DB
}

func NewGormNotificationRepository(db *gorm.DB) ports.NotificationRepository {
	return &GormNotificationRepository{db: db}
}

func (r *GormNotificationRepository) Create(ctx context.Context, notif *entities.Notification) error {
	if err := r.db.WithContext(ctx).Create(notif).Error; err != nil {
		return exceptions.NewInternalError("failed to record notification", err)
	}
	return nil
}

func (r *GormNotificationRepository) FindByUserID(ctx context.Context, userID string) ([]entities.Notification, error) {
	var notifs []entities.Notification
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&notifs).Error; err != nil {
		return nil, exceptions.NewInternalError("failed to list notifications", err)
	}
	return notifs, nil
}
