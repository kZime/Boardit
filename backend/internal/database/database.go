package database

import (
	"fmt"
	"os"
	"strings"

	"backend/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init initializes the database connection and auto-migrates all models.
// If DATABASE_DSN is ":memory:" or contains "sqlite", SQLite is used (no setup needed for tests).
// Otherwise PostgreSQL is used (e.g. production or CI).
func Init() error {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		return fmt.Errorf("DATABASE_DSN is not set")
	}

	var dialector gorm.Dialector
	if dsn == ":memory:" || strings.Contains(dsn, "sqlite") {
		sqliteDSN := dsn
		if dsn == ":memory:" {
			sqliteDSN = "file::memory:?cache=shared"
		}
		dialector = sqlite.Open(sqliteDSN)
	} else {
		dialector = postgres.Open(dsn)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate all models in dependency order
	if err := db.AutoMigrate(
		&model.User{},         // 1. User table (depended on by others)
		&model.Folder{},       // 2. Folder table (depends on user)
		&model.Note{},         // 3. Note table (depends on user and folder)
		&model.NoteRevision{}, // 4. Note revision table (depends on note)
	); err != nil {
		return fmt.Errorf("failed to auto migrate models: %w", err)
	}

	DB = db
	return nil
}

// TruncateAllTables deletes all data from tables.
// On PostgreSQL uses TRUNCATE ... RESTART IDENTITY CASCADE for reliable test isolation.
// On SQLite uses DELETE (TRUNCATE not supported).
func TruncateAllTables() error {
	if DB == nil {
		return nil
	}
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == ":memory:" || strings.Contains(dsn, "sqlite") {
		// SQLite: delete in dependency order
		for _, table := range []string{"note_revisions", "notes", "folders", "users"} {
			if err := DB.Exec("DELETE FROM " + table).Error; err != nil {
				return err
			}
		}
		return nil
	}
	// PostgreSQL: single TRUNCATE with CASCADE and identity reset (public schema for CI/macOS)
	return DB.Exec("TRUNCATE TABLE public.note_revisions, public.notes, public.folders, public.users RESTART IDENTITY CASCADE").Error
}
