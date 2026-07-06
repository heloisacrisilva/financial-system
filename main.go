package main

import (
	"financial/system/api/config"
	"log"

	"github.com/joho/godotenv"
)

var (
	logger *config.Logger
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, loading environment variables from system.")
	}

	err := config.Init()
	if err != nil {
		log.Fatalf("config initialization error: %v", err)
	}

	logger = config.GetLogger()

	logger.Info("Starting server...")
}
