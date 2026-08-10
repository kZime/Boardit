package noteapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend/internal/database"
	"backend/internal/model"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestService(t *testing.T) (*Service, *gorm.DB, model.User, model.User) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	now := time.Now()
	users := []model.User{
		{Username: "writer-one", Email: "one@example.com", PasswordHash: "test", CreatedAt: now, UpdatedAt: now},
		{Username: "writer-two", Email: "two@example.com", PasswordHash: "test", CreatedAt: now, UpdatedAt: now},
	}
	require.NoError(t, db.Create(&users).Error)
	return NewService(NewGormRepository(db)), db, users[0], users[1]
}

func TestServiceCreatesVersionedRevisionsAndOutboxEvents(t *testing.T) {
	service, db, owner, other := newTestService(t)
	ctx := context.Background()
	created, err := service.CreateNote(ctx, owner.ID, CreateNoteInput{Title: "Versioned", ContentMD: "v1"})
	require.NoError(t, err)
	require.Equal(t, uint64(1), created.Version)

	content := "v2"
	createdVersion := created.Version
	updated, err := service.UpdateNote(ctx, owner.ID, created.ID, UpdateNoteInput{
		ContentMD: &content, Version: &createdVersion, UpdatedAt: created.UpdatedAt,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(2), updated.Version)

	revisions, err := service.ListNoteRevisions(ctx, owner.ID, created.ID)
	require.NoError(t, err)
	require.Len(t, revisions, 2)
	require.Equal(t, uint64(2), revisions[0].Version)
	require.Equal(t, "v2", revisions[0].ContentMD)

	_, err = service.ListNoteRevisions(ctx, other.ID, created.ID)
	var useCaseErr *UseCaseError
	require.True(t, errors.As(err, &useCaseErr))
	require.Equal(t, ErrorNotFound, useCaseErr.Kind)

	var eventCount int64
	require.NoError(t, db.Model(&model.OutboxEvent{}).
		Where("aggregate_id = ? AND user_id = ?", created.ID, owner.ID).Count(&eventCount).Error)
	require.Equal(t, int64(2), eventCount)
}

func TestRevisionFailureRollsBackNoteUpdate(t *testing.T) {
	service, db, owner, _ := newTestService(t)
	ctx := context.Background()
	created, err := service.CreateNote(ctx, owner.ID, CreateNoteInput{Title: "Atomic", ContentMD: "v1"})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TRIGGER reject_second_revision
		BEFORE INSERT ON note_revisions WHEN NEW.version = 2
		BEGIN SELECT RAISE(FAIL, 'revision rejected'); END`).Error)

	content := "must roll back"
	createdVersion := created.Version
	_, err = service.UpdateNote(ctx, owner.ID, created.ID, UpdateNoteInput{
		ContentMD: &content, Version: &createdVersion, UpdatedAt: created.UpdatedAt,
	})
	require.Error(t, err)

	stored, err := service.GetNote(ctx, owner.ID, created.ID)
	require.NoError(t, err)
	require.Equal(t, "v1", stored.ContentMD)
	require.Equal(t, uint64(1), stored.Version)
}

func TestServiceRejectsUpdateWithoutVersion(t *testing.T) {
	service, _, owner, _ := newTestService(t)
	created, err := service.CreateNote(context.Background(), owner.ID, CreateNoteInput{Title: "Version required"})
	require.NoError(t, err)
	title := "unsafe update"

	_, err = service.UpdateNote(context.Background(), owner.ID, created.ID, UpdateNoteInput{Title: &title})
	var useCaseErr *UseCaseError
	require.True(t, errors.As(err, &useCaseErr))
	require.Equal(t, ErrorInvalid, useCaseErr.Kind)
}

func TestServiceUpdatesNoteWithoutGin(t *testing.T) {
	service, _, owner, _ := newTestService(t)
	ctx := context.Background()
	created, err := service.CreateNote(ctx, owner.ID, CreateNoteInput{
		Title: "Service boundary", ContentMD: `<script>alert("xss")</script> **safe**`,
	})
	require.NoError(t, err)
	require.NotContains(t, created.ContentHTML, "<script>")
	require.Contains(t, created.ContentHTML, "&lt;script&gt;")

	newContent := "updated"
	createdVersion := created.Version
	updated, err := service.UpdateNote(ctx, owner.ID, created.ID, UpdateNoteInput{
		ContentMD: &newContent, Version: &createdVersion,
	})
	require.NoError(t, err)
	require.Equal(t, newContent, updated.ContentMD)

	_, err = service.UpdateNote(ctx, owner.ID, created.ID, UpdateNoteInput{
		ContentMD: &newContent, Version: &createdVersion,
	})
	var conflict *ConflictError
	require.True(t, errors.As(err, &conflict))
}

func TestServiceEnforcesFolderOwnershipWithoutGin(t *testing.T) {
	service, db, owner, other := newTestService(t)
	now := time.Now()
	foreign := model.Folder{UserID: other.ID, Name: "Foreign", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(&foreign).Error)

	_, err := service.CreateNote(context.Background(), owner.ID, CreateNoteInput{
		Title: "Blocked", FolderID: &foreign.ID,
	})
	var useCaseErr *UseCaseError
	require.True(t, errors.As(err, &useCaseErr))
	require.Equal(t, ErrorInvalid, useCaseErr.Kind)
}

func TestServicePublishesUnlistedNoteWithoutGin(t *testing.T) {
	service, _, owner, _ := newTestService(t)
	ctx := context.Background()
	created, err := service.CreateNote(ctx, owner.ID, CreateNoteInput{Title: "Share me"})
	require.NoError(t, err)
	visibility := "unlisted"
	published := true
	createdVersion := created.Version
	_, err = service.UpdateNote(ctx, owner.ID, created.ID, UpdateNoteInput{
		Visibility: &visibility, IsPublished: &published, Version: &createdVersion, UpdatedAt: created.UpdatedAt,
	})
	require.NoError(t, err)

	publicNote, err := service.GetPublicNote(ctx, owner.Username, created.Slug)
	require.NoError(t, err)
	require.Equal(t, created.ID, publicNote.ID)
	require.Equal(t, owner.Username, publicNote.AuthorUsername)
}
