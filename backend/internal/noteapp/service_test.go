package noteapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend/internal/model"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestService(t *testing.T) (*Service, *gorm.DB, model.User, model.User) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Folder{}, &model.Note{}, &model.NoteRevision{}))
	now := time.Now()
	users := []model.User{
		{Username: "writer-one", Email: "one@example.com", PasswordHash: "test", CreatedAt: now, UpdatedAt: now},
		{Username: "writer-two", Email: "two@example.com", PasswordHash: "test", CreatedAt: now, UpdatedAt: now},
	}
	require.NoError(t, db.Create(&users).Error)
	return NewService(NewGormRepository(db)), db, users[0], users[1]
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
	updated, err := service.UpdateNote(ctx, owner.ID, created.ID, UpdateNoteInput{
		ContentMD: &newContent, UpdatedAt: created.UpdatedAt,
	})
	require.NoError(t, err)
	require.Equal(t, newContent, updated.ContentMD)

	_, err = service.UpdateNote(ctx, owner.ID, created.ID, UpdateNoteInput{
		ContentMD: &newContent, UpdatedAt: created.UpdatedAt,
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
	_, err = service.UpdateNote(ctx, owner.ID, created.ID, UpdateNoteInput{
		Visibility: &visibility, IsPublished: &published, UpdatedAt: created.UpdatedAt,
	})
	require.NoError(t, err)

	publicNote, err := service.GetPublicNote(ctx, owner.Username, created.Slug)
	require.NoError(t, err)
	require.Equal(t, created.ID, publicNote.ID)
	require.Equal(t, owner.Username, publicNote.AuthorUsername)
}
