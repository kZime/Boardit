package database

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

//go:embed migrations/*/*.sql
var migrationFiles embed.FS

type migration struct {
	version  int64
	name     string
	up       string
	down     string
	checksum string
}

func migrationDialect(db *gorm.DB) (string, error) {
	switch db.Dialector.Name() {
	case "postgres":
		return "postgres", nil
	case "sqlite":
		return "sqlite", nil
	default:
		return "", fmt.Errorf("unsupported migration dialect %q", db.Dialector.Name())
	}
}

func loadMigrations(dialect string) ([]migration, error) {
	directory := path.Join("migrations", dialect)
	entries, err := fs.ReadDir(migrationFiles, directory)
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	byVersion := make(map[int64]*migration)
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".up.sql") && !strings.HasSuffix(entry.Name(), ".down.sql")) {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration name %q", entry.Name())
		}
		version, parseErr := strconv.ParseInt(parts[0], 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid migration version in %q: %w", entry.Name(), parseErr)
		}
		contents, readErr := migrationFiles.ReadFile(path.Join(directory, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("read migration %q: %w", entry.Name(), readErr)
		}
		item := byVersion[version]
		if item == nil {
			item = &migration{version: version, name: strings.TrimSuffix(strings.TrimSuffix(parts[1], ".up.sql"), ".down.sql")}
			byVersion[version] = item
		}
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			item.up = string(contents)
			item.checksum = fmt.Sprintf("%x", sha256.Sum256(contents))
		} else {
			item.down = string(contents)
		}
	}
	result := make([]migration, 0, len(byVersion))
	for _, item := range byVersion {
		if item.up == "" || item.down == "" {
			return nil, fmt.Errorf("migration %d must have up and down SQL", item.version)
		}
		result = append(result, *item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].version < result[j].version })
	return result, nil
}

func ensureMigrationTable(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
        version BIGINT PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        checksum VARCHAR(64) NOT NULL,
        applied_at TIMESTAMP NOT NULL
    )`).Error
}

func applyMigrations(db *gorm.DB, migrations []migration) error {
	var applied []struct {
		Version  int64
		Checksum string
	}
	if err := db.Table("schema_migrations").Select("version", "checksum").Find(&applied).Error; err != nil {
		return fmt.Errorf("list applied migrations: %w", err)
	}
	appliedSet := make(map[int64]string, len(applied))
	for _, item := range applied {
		appliedSet[item.Version] = item.Checksum
	}

	for _, item := range migrations {
		if checksum, exists := appliedSet[item.version]; exists {
			if checksum != item.checksum {
				return fmt.Errorf("migration %06d_%s checksum changed after application", item.version, item.name)
			}
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(item.up).Error; err != nil {
				return err
			}
			return tx.Exec(
				"INSERT INTO schema_migrations (version, name, checksum, applied_at) VALUES (?, ?, ?, ?)",
				item.version, item.name, item.checksum, time.Now().UTC(),
			).Error
		}); err != nil {
			return fmt.Errorf("apply migration %06d_%s: %w", item.version, item.name, err)
		}
	}
	return nil
}

// Migrate applies every pending migration exactly once, in version order.
func Migrate(db *gorm.DB) error {
	dialect, err := migrationDialect(db)
	if err != nil {
		return err
	}
	migrations, err := loadMigrations(dialect)
	if err != nil {
		return err
	}
	if dialect == "postgres" {
		return db.Transaction(func(tx *gorm.DB) error {
			if err := ensureMigrationTable(tx); err != nil {
				return fmt.Errorf("create schema_migrations: %w", err)
			}
			if err := tx.Exec("LOCK TABLE schema_migrations IN EXCLUSIVE MODE").Error; err != nil {
				return fmt.Errorf("lock schema_migrations: %w", err)
			}
			return applyMigrations(tx, migrations)
		})
	}
	if err := ensureMigrationTable(db); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return applyMigrations(db, migrations)
}

// RollbackLast reverses the most recently applied migration. It is intended for
// controlled operator use; application startup only migrates forward.
func RollbackLast(db *gorm.DB) error {
	dialect, err := migrationDialect(db)
	if err != nil {
		return err
	}
	if err := ensureMigrationTable(db); err != nil {
		return err
	}
	var version int64
	result := db.Table("schema_migrations").Select("version").Order("version DESC").Limit(1).Scan(&version)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	migrations, err := loadMigrations(dialect)
	if err != nil {
		return err
	}
	for _, item := range migrations {
		if item.version != version {
			continue
		}
		return db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(item.down).Error; err != nil {
				return err
			}
			return tx.Exec("DELETE FROM schema_migrations WHERE version = ?", version).Error
		})
	}
	return fmt.Errorf("down migration for version %d not found", version)
}
