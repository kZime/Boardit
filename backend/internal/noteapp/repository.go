package noteapp

import (
	"context"
	"errors"

	"backend/internal/model"

	"gorm.io/gorm"
)

var ErrRepositoryNotFound = errors.New("repository record not found")

type Repository interface {
	ListNotes(context.Context, uint, ListNotesFilter) ([]model.Note, int64, error)
	FindNote(context.Context, uint, uint) (model.Note, error)
	CreateNote(context.Context, *model.Note) error
	SaveNote(context.Context, *model.Note) error
	DeleteNote(context.Context, *model.Note) error
	SlugExists(context.Context, uint, string, *uint) (bool, error)

	ListFolders(context.Context, uint) ([]model.Folder, error)
	FindFolder(context.Context, uint, uint) (model.Folder, error)
	CreateFolder(context.Context, *model.Folder) error
	SaveFolder(context.Context, *model.Folder) error
	DeleteFolder(context.Context, *model.Folder) error
	CountFolderContents(context.Context, uint, uint) (int64, int64, error)

	ListPublicNotes(context.Context, int, int) ([]model.Note, int64, error)
	FindPublicNote(context.Context, uint, string) (model.Note, error)
	FindUserByUsername(context.Context, string) (model.User, error)
	UsernamesByID(context.Context, []uint) (map[uint]string, error)

	WithinTransaction(context.Context, func(Repository) error) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func translateNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrRepositoryNotFound
	}
	return err
}

func (repository *GormRepository) ListNotes(ctx context.Context, userID uint, filter ListNotesFilter) ([]model.Note, int64, error) {
	query := repository.db.WithContext(ctx).Where("user_id = ?", userID)
	if filter.FolderID != "" {
		query = query.Where("folder_id = ?", filter.FolderID)
	}
	if filter.Query != "" {
		search := "%" + filter.Query + "%"
		query = query.Where("title LIKE ? OR content_md LIKE ?", search, search)
	}
	if filter.Status == "published" {
		query = query.Where("is_published = ?", true)
	} else if filter.Status == "draft" {
		query = query.Where("is_published = ?", false)
	}

	var total int64
	if err := query.Model(&model.Note{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var notes []model.Note
	if err := query.Order("created_at DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&notes).Error; err != nil {
		return nil, 0, err
	}
	return notes, total, nil
}

func (repository *GormRepository) FindNote(ctx context.Context, userID, noteID uint) (model.Note, error) {
	var note model.Note
	err := repository.db.WithContext(ctx).Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error
	return note, translateNotFound(err)
}

func (repository *GormRepository) CreateNote(ctx context.Context, note *model.Note) error {
	return repository.db.WithContext(ctx).Create(note).Error
}

func (repository *GormRepository) SaveNote(ctx context.Context, note *model.Note) error {
	return repository.db.WithContext(ctx).Save(note).Error
}

func (repository *GormRepository) DeleteNote(ctx context.Context, note *model.Note) error {
	return repository.db.WithContext(ctx).Delete(note).Error
}

func (repository *GormRepository) SlugExists(ctx context.Context, userID uint, slug string, excludeNoteID *uint) (bool, error) {
	query := repository.db.WithContext(ctx).Model(&model.Note{}).Where("user_id = ? AND slug = ?", userID, slug)
	if excludeNoteID != nil {
		query = query.Where("id != ?", *excludeNoteID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (repository *GormRepository) ListFolders(ctx context.Context, userID uint) ([]model.Folder, error) {
	var folders []model.Folder
	err := repository.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("sort_order asc, name asc").Find(&folders).Error
	return folders, err
}

func (repository *GormRepository) FindFolder(ctx context.Context, userID, folderID uint) (model.Folder, error) {
	var folder model.Folder
	err := repository.db.WithContext(ctx).Where("id = ? AND user_id = ?", folderID, userID).First(&folder).Error
	return folder, translateNotFound(err)
}

func (repository *GormRepository) CreateFolder(ctx context.Context, folder *model.Folder) error {
	return repository.db.WithContext(ctx).Create(folder).Error
}

func (repository *GormRepository) SaveFolder(ctx context.Context, folder *model.Folder) error {
	return repository.db.WithContext(ctx).Save(folder).Error
}

func (repository *GormRepository) DeleteFolder(ctx context.Context, folder *model.Folder) error {
	return repository.db.WithContext(ctx).Delete(folder).Error
}

func (repository *GormRepository) CountFolderContents(ctx context.Context, userID, folderID uint) (int64, int64, error) {
	var subfolderCount int64
	if err := repository.db.WithContext(ctx).Model(&model.Folder{}).
		Where("parent_id = ? AND user_id = ?", folderID, userID).Count(&subfolderCount).Error; err != nil {
		return 0, 0, err
	}
	var noteCount int64
	if err := repository.db.WithContext(ctx).Model(&model.Note{}).
		Where("folder_id = ? AND user_id = ?", folderID, userID).Count(&noteCount).Error; err != nil {
		return 0, 0, err
	}
	return subfolderCount, noteCount, nil
}

func (repository *GormRepository) ListPublicNotes(ctx context.Context, limit, offset int) ([]model.Note, int64, error) {
	query := repository.db.WithContext(ctx).Where("visibility = ? AND is_published = ?", "public", true)
	var total int64
	if err := query.Model(&model.Note{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var notes []model.Note
	if err := query.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&notes).Error; err != nil {
		return nil, 0, err
	}
	return notes, total, nil
}

func (repository *GormRepository) FindPublicNote(ctx context.Context, userID uint, slug string) (model.Note, error) {
	var note model.Note
	err := repository.db.WithContext(ctx).
		Where("user_id = ? AND slug = ? AND visibility IN ? AND is_published = ?", userID, slug, []string{"public", "unlisted"}, true).
		First(&note).Error
	return note, translateNotFound(err)
}

func (repository *GormRepository) FindUserByUsername(ctx context.Context, username string) (model.User, error) {
	var user model.User
	err := repository.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	return user, translateNotFound(err)
}

func (repository *GormRepository) UsernamesByID(ctx context.Context, userIDs []uint) (map[uint]string, error) {
	if len(userIDs) == 0 {
		return map[uint]string{}, nil
	}
	var users []struct {
		ID       uint
		Username string
	}
	if err := repository.db.WithContext(ctx).Model(&model.User{}).
		Where("id IN ?", userIDs).Select("id", "username").Find(&users).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]string, len(users))
	for _, user := range users {
		result[user.ID] = user.Username
	}
	return result, nil
}

func (repository *GormRepository) WithinTransaction(ctx context.Context, operation func(Repository) error) error {
	return repository.db.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		return operation(&GormRepository{db: transaction})
	})
}
