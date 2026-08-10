package database

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"testing"
	"time"

	"backend/internal/testutils"
)

type legacyUser struct {
	ID           uint `gorm:"primaryKey"`
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (legacyUser) TableName() string { return "users" }

type legacyFolder struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	Name      string
	ParentID  *uint
	SortOrder int
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (legacyFolder) TableName() string { return "folders" }

type legacyNote struct {
	ID          uint `gorm:"primaryKey"`
	UserID      uint
	FolderID    *uint
	Title       string
	Slug        string
	CoverURL    string
	ContentMd   string
	ContentHtml string
	IsPublished bool
	Visibility  string
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (legacyNote) TableName() string { return "notes" }

type legacyRevision struct {
	ID        uint `gorm:"primaryKey"`
	NoteID    uint
	ContentMd string
	Diff      *string
	CreatedAt time.Time
}

func (legacyRevision) TableName() string { return "note_revisions" }

func TestMain(m *testing.M) {
	// Load environment variables for testing
	testutils.LoadTestEnv()

	// Run test
	code := m.Run()

	// Exit test
	os.Exit(code)
}

func TestMigrateIsVersionedAndIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	require.NoError(t, Migrate(db))

	var versions int64
	require.NoError(t, db.Table("schema_migrations").Count(&versions).Error)
	require.Equal(t, int64(2), versions)
	var missingChecksums int64
	require.NoError(t, db.Table("schema_migrations").Where("checksum = ''").Count(&missingChecksums).Error)
	require.Zero(t, missingChecksums)
	require.True(t, db.Migrator().HasColumn("notes", "version"))
	for _, table := range []string{"note_revisions", "refresh_sessions", "outbox_events", "background_jobs", "ai_runs", "ai_candidates"} {
		require.Truef(t, db.Migrator().HasTable(table), "expected table %s", table)
	}
}

func TestRollbackLastMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	require.NoError(t, RollbackLast(db))

	require.False(t, db.Migrator().HasTable("ai_candidates"))
	require.False(t, db.Migrator().HasColumn("notes", "version"))
	var versions int64
	require.NoError(t, db.Table("schema_migrations").Count(&versions).Error)
	require.Equal(t, int64(1), versions)
}

func TestMigrateUpgradesLegacyAutoMigrateSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&legacyUser{}, &legacyFolder{}, &legacyNote{}, &legacyRevision{}))
	now := time.Now().UTC()
	user := legacyUser{Username: "legacy", Email: "legacy@example.com", PasswordHash: "test", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(&user).Error)
	note := legacyNote{UserID: user.ID, Title: "Legacy", Slug: "legacy", ContentMd: "old", ContentHtml: "old", Visibility: "private", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(&note).Error)
	require.NoError(t, db.Create(&legacyRevision{NoteID: note.ID, ContentMd: "old", CreatedAt: now}).Error)

	require.NoError(t, Migrate(db))
	var upgraded struct {
		UserID  uint
		Version uint64
		Title   string
	}
	require.NoError(t, db.Table("note_revisions").First(&upgraded).Error)
	require.Equal(t, user.ID, upgraded.UserID)
	require.Equal(t, uint64(1), upgraded.Version)
	require.Equal(t, note.Title, upgraded.Title)
}

func TestInit(t *testing.T) {
	// Run init function
	err := Init()

	// Check if there was an error
	assert.NoError(t, err, "Init should not return an error")

	// Check if DB is initialized
	assert.NotNil(t, DB, "DB should not be nil after initialization")
}
