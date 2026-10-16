package persistence

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/myavarih/sre-hands-on/pkg/entities"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
	"gorm.io/plugin/opentelemetry/tracing"
)

// Deterministic unique 64-bit advisory lock key for coordinating database migrations fleet-wide
const migrationAdvisoryLockKey = 748291038592019482

// NewPostgresDB connects to PostgreSQL with tuned connection pooling and schema migrations.
func NewPostgresDB(dsn string, logger *slog.Logger) (*gorm.DB, error) {
	return NewPostgresDBWithReplica(dsn, "", logger)
}

// NewPostgresDBWithReplica connects to primary and optional read replica (Flaw F-23 fix)
func NewPostgresDBWithReplica(dsn string, roDSN string, logger *slog.Logger) (*gorm.DB, error) {
	if dsn == "" {
		dsn = "host=localhost user=app_user password=app_password dbname=app_db port=5432 sslmode=disable"
	}

	gormConfig := &gorm.Config{
		Logger:         gormlogger.Default.LogMode(gormlogger.Warn),
		TranslateError: true,
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres primary: %w", err)
	}

	// Register OpenTelemetry tracing plugin for SQL query spans (Tempo / Grafana)
	if err := db.Use(tracing.NewPlugin(
		tracing.WithoutQueryVariables(), // Redact query arguments to prevent PII leakage
	)); err != nil {
		logger.Warn("Failed to register OpenTelemetry tracing plugin on GORM", "error", err)
	}

	// Register read replica with dbresolver for read/write splitting (Flaw F-23 fix)
	if roDSN != "" && roDSN != dsn {
		err := db.Use(dbresolver.Register(dbresolver.Config{
			Replicas: []gorm.Dialector{postgres.Open(roDSN)},
			Policy:   dbresolver.RandomPolicy{},
		}).
			SetMaxOpenConns(25).
			SetMaxIdleConns(10).
			SetConnMaxLifetime(5 * time.Minute).
			SetConnMaxIdleTime(2 * time.Minute))
		if err != nil {
			logger.Warn("Failed to register read replica, using primary for all queries", "error", err)
		} else {
			logger.Info("Configured database read-replica splitting with pg-cluster-ro")
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// SRE tuned connection pooling on primary
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	// Execute migrations and initial seeding under a distributed PostgreSQL advisory lock (Flaw F-07 fix)
	if err := RunWithMigrationLock(db, logger); err != nil {
		return nil, fmt.Errorf("migration lock execution failed: %w", err)
	}

	return db, nil
}

// RunWithMigrationLock executes schema migrations and initial seeding under a PostgreSQL advisory lock.
// This prevents concurrent DDL lock deadlocks and duplicate seed rows when multiple microservices start up simultaneously.
func RunWithMigrationLock(db *gorm.DB, logger *slog.Logger) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Acquire a dedicated connection from the pool so lock and unlock share the same session
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire dedicated connection for migration: %w", err)
	}
	defer conn.Close()

	// 1. Acquire session-level advisory lock (blocks until available)
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationAdvisoryLockKey); err != nil {
		return fmt.Errorf("failed to acquire migration advisory lock: %w", err)
	}
	defer func() {
		unlockCtx, unlockCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer unlockCancel()
		if _, err := conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", migrationAdvisoryLockKey); err != nil {
			logger.Warn("Failed to release migration advisory lock", "error", err)
		}
	}()

	// 2. Safely execute schema migration and seed while holding the exclusive lock
	if err := AutoMigrate(db); err != nil {
		return fmt.Errorf("failed to auto-migrate schemas: %w", err)
	}

	if err := SeedCatalogIfEmpty(db); err != nil {
		logger.Warn("Failed to seed initial catalog", "error", err)
	}

	return nil
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
