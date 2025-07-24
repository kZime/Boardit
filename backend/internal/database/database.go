package database

import (
	"fmt"
	"os"

	"backend/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init: initialize the database connection and auto migrate the user model
func Init() error {

	// use environment variable to configure DSN
	// PostgreSQL DSN example:
	//   host=localhost user=youruser password=yourpw dbname=note_blog port=5432 sslmode=disable TimeZone=UTC

	// get dsn from .env file
	dsn := os.Getenv("DATABASE_DSN")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate the user model
	if err := db.AutoMigrate(&model.User{}); err != nil {
		return fmt.Errorf("failed to auto migrate the user model: %w", err)
	}

	DB = db
	return nil
}
