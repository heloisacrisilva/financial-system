package main

import (
	"context"
	"financial/system/api/config"
	"financial/system/api/events"
	"financial/system/api/router"
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
	events.StartDispatcher(context.Background())
	router.Initialize()
}
