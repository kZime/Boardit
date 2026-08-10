package noteapp

import (
	"context"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"backend/internal/model"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func toNoteDTO(note model.Note) Note {
	return Note{
		ID: note.ID, UserID: note.UserID, FolderID: note.FolderID, Title: note.Title,
		Slug: note.Slug, CoverURL: note.CoverURL, ContentMD: note.ContentMd,
		ContentHTML: note.ContentHtml, IsPublished: note.IsPublished,
		Visibility: note.Visibility, SortOrder: note.SortOrder, Version: note.Version,
		CreatedAt: formatTimestamp(note.CreatedAt), UpdatedAt: formatTimestamp(note.UpdatedAt),
	}
}

func toRevisionDTO(revision model.NoteRevision) NoteRevision {
	return NoteRevision{
		ID: revision.ID, NoteID: revision.NoteID, Version: revision.Version,
		Title: revision.Title, ContentMD: revision.ContentMd, ContentHTML: revision.ContentHtml,
		Source: revision.Source, CreatedAt: formatTimestamp(revision.CreatedAt),
	}
}

func recordOutboxEvent(ctx context.Context, repository Repository, note model.Note, eventType string) error {
	event := model.OutboxEvent{
		UserID: note.UserID, AggregateType: "note", AggregateID: note.ID,
		AggregateVersion: note.Version, EventType: eventType,
		Payload: fmt.Sprintf(`{"note_id":%d,"user_id":%d,"version":%d}`, note.ID, note.UserID, note.Version),
		Status:  "pending", AvailableAt: note.UpdatedAt, CreatedAt: note.UpdatedAt,
	}
	if err := repository.CreateOutboxEvent(ctx, &event); err != nil {
		return internal("failed to create note change event", err)
	}
	return nil
}

func recordNoteChange(ctx context.Context, repository Repository, note model.Note, eventType string) error {
	revision := model.NoteRevision{
		NoteID: note.ID, UserID: note.UserID, Version: note.Version, Title: note.Title,
		ContentMd: note.ContentMd, ContentHtml: note.ContentHtml, Source: "user", CreatedAt: note.UpdatedAt,
	}
	if err := repository.CreateNoteRevision(ctx, &revision); err != nil {
		return internal("failed to create note revision", err)
	}
	return recordOutboxEvent(ctx, repository, note, eventType)
}

func persistNoteChange(ctx context.Context, repository Repository, note *model.Note, expectedVersion uint64, eventType string) error {
	saved, err := repository.SaveNoteIfVersion(ctx, note, expectedVersion)
	if err != nil {
		return internal("failed to update note", err)
	}
	if !saved {
		current, findErr := repository.FindNote(ctx, note.UserID, note.ID)
		if findErr != nil {
			return internal("failed to reload conflicting note", findErr)
		}
		return &ConflictError{
			ServerUpdatedAt: formatTimestamp(current.UpdatedAt),
			ServerSnapshot:  toNoteDTO(current),
		}
	}
	return recordNoteChange(ctx, repository, *note, eventType)
}

func toFolderDTO(folder model.Folder) Folder {
	return Folder{
		ID: folder.ID, UserID: folder.UserID, Name: folder.Name, ParentID: folder.ParentID,
		SortOrder: folder.SortOrder, CreatedAt: formatTimestamp(folder.CreatedAt),
		UpdatedAt: formatTimestamp(folder.UpdatedAt),
	}
}

func normalizePage(limit, offset int) (int, int) {
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (service *Service) ListNotes(ctx context.Context, userID uint, filter ListNotesFilter) (NotePage, error) {
	filter.Limit, filter.Offset = normalizePage(filter.Limit, filter.Offset)
	notes, total, err := service.repository.ListNotes(ctx, userID, filter)
	if err != nil {
		return NotePage{}, internal("failed to list notes", err)
	}
	items := make([]Note, 0, len(notes))
	for _, note := range notes {
		items = append(items, toNoteDTO(note))
	}
	return NotePage{Items: items, Total: total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func (service *Service) GetNote(ctx context.Context, userID, noteID uint) (Note, error) {
	note, err := service.repository.FindNote(ctx, userID, noteID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return Note{}, notFound("note not found")
	}
	if err != nil {
		return Note{}, internal("failed to get note", err)
	}
	return toNoteDTO(note), nil
}

func (service *Service) requireFolder(ctx context.Context, userID uint, folderID *uint) error {
	if folderID == nil {
		return nil
	}
	if _, err := service.repository.FindFolder(ctx, userID, *folderID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return invalid("folder not found or access denied")
		}
		return internal("failed to validate folder", err)
	}
	return nil
}

func (service *Service) uniqueSlug(ctx context.Context, title string, userID uint, excludeNoteID *uint) (string, error) {
	base := generateSlug(title)
	exists, err := service.repository.SlugExists(ctx, userID, base, excludeNoteID)
	if err != nil {
		return "", internal("failed to generate note slug", err)
	}
	if !exists {
		return base, nil
	}
	for suffix := 2; suffix <= 999; suffix++ {
		candidate := fmt.Sprintf("%s-%d", base, suffix)
		exists, err = service.repository.SlugExists(ctx, userID, candidate, excludeNoteID)
		if err != nil {
			return "", internal("failed to generate note slug", err)
		}
		if !exists {
			return candidate, nil
		}
	}
	return fmt.Sprintf("%s-%d", base, time.Now().Unix()), nil
}

func (service *Service) CreateNote(ctx context.Context, userID uint, input CreateNoteInput) (Note, error) {
	if err := service.requireFolder(ctx, userID, input.FolderID); err != nil {
		return Note{}, err
	}
	if input.Title == "" {
		input.Title = "Untitled"
	}
	if input.ContentMD == "" {
		input.ContentMD = "# New note"
	}
	slug, err := service.uniqueSlug(ctx, input.Title, userID, nil)
	if err != nil {
		return Note{}, err
	}
	now := nowTimestamp()
	note := model.Note{
		UserID: userID, FolderID: input.FolderID, Title: input.Title, Slug: slug,
		ContentMd: input.ContentMD, ContentHtml: convertMarkdownToHTML(input.ContentMD),
		Visibility: "private", Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := service.repository.WithinTransaction(ctx, func(transaction Repository) error {
		if err := transaction.CreateNote(ctx, &note); err != nil {
			return internal("failed to create note", err)
		}
		return recordNoteChange(ctx, transaction, note, "note.created")
	}); err != nil {
		return Note{}, err
	}
	return toNoteDTO(note), nil
}

func validVisibility(visibility string) bool {
	return visibility == "private" || visibility == "public" || visibility == "unlisted"
}

func (service *Service) UpdateNote(ctx context.Context, userID, noteID uint, input UpdateNoteInput) (Note, error) {
	var updated Note
	err := service.repository.WithinTransaction(ctx, func(transaction Repository) error {
		var updateErr error
		updated, updateErr = NewService(transaction).updateNoteInTransaction(ctx, userID, noteID, input)
		return updateErr
	})
	if err != nil {
		return Note{}, err
	}
	return updated, nil
}

func (service *Service) updateNoteInTransaction(ctx context.Context, userID, noteID uint, input UpdateNoteInput) (Note, error) {
	note, err := service.repository.FindNote(ctx, userID, noteID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return Note{}, notFound("note not found")
	}
	if err != nil {
		return Note{}, internal("failed to get note", err)
	}
	if input.Version != nil && note.Version != *input.Version {
		return Note{}, &ConflictError{
			ServerUpdatedAt: formatTimestamp(note.UpdatedAt),
			ServerSnapshot:  toNoteDTO(note),
		}
	}
	if input.UpdatedAt != "" {
		expected, parseErr := time.Parse(time.RFC3339, input.UpdatedAt)
		if parseErr != nil {
			return Note{}, invalid("invalid updated_at format")
		}
		if !note.UpdatedAt.Equal(expected) {
			return Note{}, &ConflictError{
				ServerUpdatedAt: formatTimestamp(note.UpdatedAt),
				ServerSnapshot:  toNoteDTO(note),
			}
		}
	}

	hasChanges := false
	if input.Title != nil {
		note.Title = *input.Title
		note.Slug, err = service.uniqueSlug(ctx, note.Title, userID, &noteID)
		if err != nil {
			return Note{}, err
		}
		hasChanges = true
	}
	if input.FolderID.Set {
		if err := service.requireFolder(ctx, userID, input.FolderID.Value); err != nil {
			return Note{}, err
		}
		note.FolderID = input.FolderID.Value
		hasChanges = true
	}
	if input.CoverURL != nil && *input.CoverURL != note.CoverURL {
		note.CoverURL = *input.CoverURL
		hasChanges = true
	}
	if input.ContentMD != nil && *input.ContentMD != note.ContentMd {
		note.ContentMd = *input.ContentMD
		note.ContentHtml = convertMarkdownToHTML(note.ContentMd)
		hasChanges = true
	}
	if input.IsPublished != nil {
		note.IsPublished = *input.IsPublished
		hasChanges = true
	}
	if input.Visibility != nil {
		if !validVisibility(*input.Visibility) {
			return Note{}, invalid("invalid visibility")
		}
		note.Visibility = *input.Visibility
		hasChanges = true
	}
	if !hasChanges {
		return toNoteDTO(note), nil
	}
	expectedVersion := note.Version
	note.Version++
	note.UpdatedAt = nowTimestamp()
	if err := persistNoteChange(ctx, service.repository, &note, expectedVersion, "note.updated"); err != nil {
		return Note{}, err
	}
	return toNoteDTO(note), nil
}

func (service *Service) ListNoteRevisions(ctx context.Context, userID, noteID uint) ([]NoteRevision, error) {
	if _, err := service.repository.FindNote(ctx, userID, noteID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, notFound("note not found")
		}
		return nil, internal("failed to get note", err)
	}
	revisions, err := service.repository.ListNoteRevisions(ctx, userID, noteID)
	if err != nil {
		return nil, internal("failed to list note revisions", err)
	}
	result := make([]NoteRevision, 0, len(revisions))
	for _, revision := range revisions {
		result = append(result, toRevisionDTO(revision))
	}
	return result, nil
}

func (service *Service) DeleteNote(ctx context.Context, userID, noteID uint) error {
	return service.repository.WithinTransaction(ctx, func(transaction Repository) error {
		note, err := transaction.FindNote(ctx, userID, noteID)
		if errors.Is(err, ErrRepositoryNotFound) {
			return notFound("note not found")
		}
		if err != nil {
			return internal("failed to get note", err)
		}
		note.Version++
		note.UpdatedAt = nowTimestamp()
		if err := recordOutboxEvent(ctx, transaction, note, "note.deleted"); err != nil {
			return err
		}
		if err := transaction.DeleteNote(ctx, &note); err != nil {
			return internal("failed to delete note", err)
		}
		return nil
	})
}

func (service *Service) ListFolders(ctx context.Context, userID uint) (FolderList, error) {
	folders, err := service.repository.ListFolders(ctx, userID)
	if err != nil {
		return FolderList{}, internal("failed to list folders", err)
	}
	items := make([]Folder, 0, len(folders))
	for _, folder := range folders {
		items = append(items, toFolderDTO(folder))
	}
	return FolderList{Items: items}, nil
}

func (service *Service) CreateFolder(ctx context.Context, userID uint, input CreateFolderInput) (Folder, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Folder{}, invalid("folder name is required")
	}
	if err := service.requireFolder(ctx, userID, input.ParentID); err != nil {
		return Folder{}, err
	}
	now := nowTimestamp()
	folder := model.Folder{
		UserID: userID, Name: input.Name, ParentID: input.ParentID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := service.repository.CreateFolder(ctx, &folder); err != nil {
		return Folder{}, internal("failed to create folder", err)
	}
	return toFolderDTO(folder), nil
}

func (service *Service) folderParentWouldCycle(ctx context.Context, userID, folderID, parentID uint) (bool, error) {
	visited := map[uint]struct{}{folderID: {}}
	currentID := parentID
	for {
		if _, exists := visited[currentID]; exists {
			return true, nil
		}
		visited[currentID] = struct{}{}
		current, err := service.repository.FindFolder(ctx, userID, currentID)
		if errors.Is(err, ErrRepositoryNotFound) {
			return false, invalid("parent folder not found or access denied")
		}
		if err != nil {
			return false, internal("failed to validate parent folder", err)
		}
		if current.ParentID == nil {
			return false, nil
		}
		currentID = *current.ParentID
	}
}

func (service *Service) UpdateFolder(ctx context.Context, userID, folderID uint, input UpdateFolderInput) (Folder, error) {
	folder, err := service.repository.FindFolder(ctx, userID, folderID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return Folder{}, notFound("folder not found")
	}
	if err != nil {
		return Folder{}, internal("failed to get folder", err)
	}
	hasChanges := false
	if input.Name != "" {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			return Folder{}, invalid("folder name is required")
		}
		folder.Name = name
		hasChanges = true
	}
	if input.ParentID.Set {
		if input.ParentID.Value != nil {
			wouldCycle, err := service.folderParentWouldCycle(ctx, userID, folderID, *input.ParentID.Value)
			if err != nil {
				return Folder{}, err
			}
			if wouldCycle {
				return Folder{}, invalid("folder parent would create a cycle")
			}
		}
		folder.ParentID = input.ParentID.Value
		hasChanges = true
	}
	if !hasChanges {
		return toFolderDTO(folder), nil
	}
	folder.UpdatedAt = nowTimestamp()
	if err := service.repository.SaveFolder(ctx, &folder); err != nil {
		return Folder{}, internal("failed to update folder", err)
	}
	return toFolderDTO(folder), nil
}

func (service *Service) DeleteFolder(ctx context.Context, userID, folderID uint) error {
	folder, err := service.repository.FindFolder(ctx, userID, folderID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return notFound("folder not found")
	}
	if err != nil {
		return internal("failed to get folder", err)
	}
	subfolders, notes, err := service.repository.CountFolderContents(ctx, userID, folderID)
	if err != nil {
		return internal("failed to inspect folder contents", err)
	}
	if subfolders > 0 || notes > 0 {
		return invalid("folder is not empty")
	}
	if err := service.repository.DeleteFolder(ctx, &folder); err != nil {
		return internal("failed to delete folder", err)
	}
	return nil
}

func (service *Service) Reorder(ctx context.Context, userID uint, input ReorderInput) error {
	err := service.repository.WithinTransaction(ctx, func(transaction Repository) error {
		transactionService := NewService(transaction)
		for _, item := range input.Folders {
			folder, err := transaction.FindFolder(ctx, userID, item.ID)
			if errors.Is(err, ErrRepositoryNotFound) {
				return notFound("folder not found")
			}
			if err != nil {
				return internal("failed to get folder", err)
			}
			folder.SortOrder = item.SortOrder
			if err := transaction.SaveFolder(ctx, &folder); err != nil {
				return internal("failed to update folder", err)
			}
		}
		for _, item := range input.Notes {
			note, err := transaction.FindNote(ctx, userID, item.ID)
			if errors.Is(err, ErrRepositoryNotFound) {
				return notFound("note not found")
			}
			if err != nil {
				return internal("failed to get note", err)
			}
			if err := transactionService.requireFolder(ctx, userID, item.FolderID); err != nil {
				return err
			}
			note.SortOrder = item.SortOrder
			note.FolderID = item.FolderID
			expectedVersion := note.Version
			note.Version++
			note.UpdatedAt = nowTimestamp()
			if err := persistNoteChange(ctx, transaction, &note, expectedVersion, "note.updated"); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		var useCaseErr *UseCaseError
		if errors.As(err, &useCaseErr) {
			return useCaseErr
		}
		return internal("failed to reorder tree", err)
	}
	return nil
}

func (service *Service) ListPublicNotes(ctx context.Context, limit, offset int) (PublicNotePage, error) {
	limit, offset = normalizePage(limit, offset)
	notes, total, err := service.repository.ListPublicNotes(ctx, limit, offset)
	if err != nil {
		return PublicNotePage{}, internal("failed to list public notes", err)
	}
	userIDs := make([]uint, 0, len(notes))
	for _, note := range notes {
		userIDs = append(userIDs, note.UserID)
	}
	usernames, err := service.repository.UsernamesByID(ctx, userIDs)
	if err != nil {
		return PublicNotePage{}, internal("failed to load note authors", err)
	}
	items := make([]PublicNoteListItem, 0, len(notes))
	for _, note := range notes {
		excerpt := stripFrontmatter(note.ContentMd)
		if len([]rune(excerpt)) > 200 {
			excerpt = string([]rune(excerpt)[:200]) + "..."
		}
		items = append(items, PublicNoteListItem{
			ID: note.ID, Title: note.Title, Slug: note.Slug, UserID: note.UserID,
			AuthorUsername: usernames[note.UserID], Excerpt: excerpt, CoverURL: note.CoverURL,
			CreatedAt: formatTimestamp(note.CreatedAt), UpdatedAt: formatTimestamp(note.UpdatedAt),
		})
	}
	return PublicNotePage{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (service *Service) GetPublicNote(ctx context.Context, username, slug string) (PublicNote, error) {
	if username == "" || slug == "" {
		return PublicNote{}, invalid("username and slug required")
	}
	user, err := service.repository.FindUserByUsername(ctx, username)
	if errors.Is(err, ErrRepositoryNotFound) {
		return PublicNote{}, notFound("user not found")
	}
	if err != nil {
		return PublicNote{}, internal("failed to get user", err)
	}
	note, err := service.repository.FindPublicNote(ctx, user.ID, slug)
	if errors.Is(err, ErrRepositoryNotFound) {
		return PublicNote{}, notFound("note not found")
	}
	if err != nil {
		return PublicNote{}, internal("failed to get note", err)
	}
	return PublicNote{Note: toNoteDTO(note), AuthorUsername: user.Username}, nil
}

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = regexp.MustCompile("[^a-z0-9-]").ReplaceAllString(slug, "")
	slug = regexp.MustCompile("-+").ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "untitled"
	}
	return slug
}

func convertMarkdownToHTML(markdown string) string {
	markdown = html.EscapeString(markdown)
	converted := regexp.MustCompile(`^# (.+)$`).ReplaceAllString(markdown, "<h1>$1</h1>")
	converted = regexp.MustCompile(`^## (.+)$`).ReplaceAllString(converted, "<h2>$1</h2>")
	converted = regexp.MustCompile(`^### (.+)$`).ReplaceAllString(converted, "<h3>$1</h3>")
	converted = regexp.MustCompile(`\*\*(.+?)\*\*`).ReplaceAllString(converted, "<strong>$1</strong>")
	converted = regexp.MustCompile(`\*(.+?)\*`).ReplaceAllString(converted, "<em>$1</em>")
	return strings.ReplaceAll(converted, "\n", "<br>")
}

func stripFrontmatter(markdown string) string {
	trimmed := strings.TrimSpace(markdown)
	if !strings.HasPrefix(trimmed, "---") {
		return markdown
	}
	lines := strings.Split(trimmed, "\n")
	if strings.TrimSpace(lines[0]) != "---" {
		return markdown
	}
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			return strings.TrimSpace(strings.Join(lines[index+1:], "\n"))
		}
	}
	return markdown
}
