package config

import (
	"financial/system/api/entities"
	"fmt"
	"os"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	dbHost     string
	dbPort     string
	dbUser     string
	dbPassword string
	dbName     string
	once       sync.Once
)

func loadEnvVars() {
	once.Do(func() {
		dbHost = os.Getenv("DB_HOST")
		dbPort = os.Getenv("DB_PORT")
		dbUser = os.Getenv("DB_USER")
		dbPassword = os.Getenv("DB_PASSWORD")
		dbName = os.Getenv("DB_NAME")
	})
}

func InitializeConnectDB(logger *Logger) (*gorm.DB, error) {
	loadEnvVars()

	var db *gorm.DB
	var err error

	connStr := fmt.Sprintf("host=%s  port=%s  user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  connStr,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})

	if err != nil {
		logger.Errorf("Error while opening PostgreSQL connection: %v", err)
		return nil, err
	}

	models := []interface{}{
		&entities.Account{},
		&entities.Client{},
		&entities.OperationEvent{},
		&entities.OperationReference{},
		&entities.TransactionHistory{},
	}

	if err := db.AutoMigrate(models...); err != nil {
		logger.Errorf("Failed to migrate models: %v", err)
		return nil, err
	}

	if err := db.Exec(`
		INSERT INTO operation_references (reference_id, created_at)
		SELECT reference_id, MIN(created_at)
		FROM transaction_histories
		GROUP BY reference_id
		ON CONFLICT (reference_id) DO NOTHING
	`).Error; err != nil {
		logger.Errorf("Failed to backfill operation references: %v", err)
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Errorf("Error getting generic db object: %v", err)
		return nil, fmt.Errorf("error getting generic db object: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	logger.Info("Successfully connected to database:", dbName)

	return db, nil
}
