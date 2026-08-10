package database

import (
	"fmt"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init initializes the database connection and applies pending SQL migrations.
// If DATABASE_DSN is ":memory:" or contains "sqlite", SQLite is used (no setup needed for tests).
// Otherwise PostgreSQL is used (e.g. production or CI).
func Init() error {
	return InitWithDSN(os.Getenv("DATABASE_DSN"))
}

func InitWithDSN(dsn string) error {
	db, err := OpenWithDSN(dsn)
	if err != nil {
		return err
	}

	if err := Migrate(db); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	DB = db
	return nil
}

// OpenWithDSN opens a supported database without changing its schema.
func OpenWithDSN(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_DSN is not set")
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
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}

// TruncateAllTables deletes all data from tables.
// On PostgreSQL uses TRUNCATE ... RESTART IDENTITY CASCADE for reliable test isolation.
// On SQLite uses DELETE (TRUNCATE not supported).
func TruncateAllTables() error {
	if DB == nil {
		return nil
	}
	if DB.Dialector.Name() == "sqlite" {
		// SQLite: delete in dependency order
		for _, table := range []string{"ai_candidates", "ai_runs", "background_jobs", "outbox_events", "refresh_sessions", "note_revisions", "notes", "folders", "users"} {
			if err := DB.Exec("DELETE FROM " + table).Error; err != nil {
				return err
			}
		}
		return nil
	}
	// PostgreSQL: single TRUNCATE with CASCADE and identity reset (public schema for CI/macOS)
	return DB.Exec("TRUNCATE TABLE public.ai_candidates, public.ai_runs, public.background_jobs, public.outbox_events, public.refresh_sessions, public.note_revisions, public.notes, public.folders, public.users RESTART IDENTITY CASCADE").Error
}
