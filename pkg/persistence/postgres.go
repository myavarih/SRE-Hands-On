package persistence

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/myavarih/sre-hands-on/pkg/entities"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// NewPostgresDB connects to PostgreSQL with tuned connection pooling and schema migrations.
func NewPostgresDB(dsn string, logger *slog.Logger) (*gorm.DB, error) {
	if dsn == "" {
		dsn = "host=localhost user=app_user password=app_password dbname=app_db port=5432 sslmode=disable"
	}

	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// SRE tuned connection pooling
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	if err := AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate schemas: %w", err)
	}

	if err := SeedCatalogIfEmpty(db); err != nil {
		logger.Warn("Failed to seed initial catalog", "error", err)
	}

	return db, nil
}

// AutoMigrate executes schema migrations for all entities.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&entities.User{},
		&entities.Product{},
		&entities.Order{},
		&entities.OrderItem{},
		&entities.Payment{},
		&entities.Notification{},
	)
}

// SeedCatalogIfEmpty populates initial items if the products table has zero rows.
func SeedCatalogIfEmpty(db *gorm.DB) error {
	var count int64
	if err := db.Model(&entities.Product{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	products := []entities.Product{
		{
			Name:          "Cloud-Native Developer Laptop Pro",
			Description:   "16-inch M3 Max 64GB RAM high-performance development workstation",
			PriceCents:    249900,
			StockQuantity: 50,
		},
		{
			Name:          "Ergonomic Split Mechanical Keyboard",
			Description:   "Low-profile hot-swappable mechanical switches with custom layout",
			PriceCents:    18900,
			StockQuantity: 100,
		},
		{
			Name:          "Ultra-Wide 49-inch Curved Monitor",
			Description:   "Dual QHD 144Hz 5120x1440 panoramic productivity display",
			PriceCents:    89900,
			StockQuantity: 30,
		},
		{
			Name:          "Active Noise-Canceling SRE Headphones",
			Description:   "Lossless audio with ultra-low latency and 40-hour battery life",
			PriceCents:    29900,
			StockQuantity: 75,
		},
	}

	return db.Create(&products).Error
}
