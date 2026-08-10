package main

import (
	"backend/internal/config"
	"backend/internal/database"
	"log"
	"os"

	"backend/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	// Load .env file if present (e.g. local dev); in Docker env vars are set by the runtime
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatal("Error loading .env file: ", err)
	}

	configuration, err := config.Load()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	// Init database
	if err := database.InitWithDSN(configuration.DatabaseDSN); err != nil {
		log.Fatalf("failed to initialize the database: %v", err)
	}

	// Production: set GIN_MODE=release to disable debug and trust only configured proxies
	if configuration.GinMode != "" {
		gin.SetMode(configuration.GinMode)
	}

	// Init router
	r := router.SetupWithOptions(database.DB, router.Options{
		JWTSecret:      configuration.JWTSecret,
		CORSOrigins:    configuration.CORSOrigins,
		TrustedProxies: configuration.TrustedProxies,
	})
	if err := r.Run(configuration.ListenAddress); err != nil {
		log.Fatalf("server failed: %v", err)
	}

}
