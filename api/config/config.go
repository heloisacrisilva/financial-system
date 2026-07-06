package config

import (
	"sync"

	"gorm.io/gorm"
)

var (
	db       *gorm.DB
	logger   *Logger
	initOnce sync.Once
	initErr  error
)

func Init() error {
	initOnce.Do(func() {
		InitLogger("[Financial System] ")
		db, initErr = InitializeConnectDB(logger)
	})
	return initErr
}

func InitLogger(p string) {
	logger = NewLogger(p)
}

func GetPostgres() *gorm.DB {
	return db
}

func GetLogger() *Logger {
	return logger
}
