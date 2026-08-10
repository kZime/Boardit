package noteapp

import (
	"encoding/json"
	"time"
)

type OptionalUint struct {
	Set   bool
	Value *uint
}

func (value *OptionalUint) UnmarshalJSON(data []byte) error {
	value.Set = true
	if string(data) == "null" {
		value.Value = nil
		return nil
	}
	var parsed uint
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	value.Value = &parsed
	return nil
}

type Note struct {
	ID          uint   `json:"id"`
	UserID      uint   `json:"user_id"`
	FolderID    *uint  `json:"folder_id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	CoverURL    string `json:"cover_url"`
	ContentMD   string `json:"content_md"`
	ContentHTML string `json:"content_html"`
	IsPublished bool   `json:"is_published"`
	Visibility  string `json:"visibility"`
	SortOrder   int    `json:"sort_order"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type NotePage struct {
	Items  []Note `json:"items"`
	Total  int64  `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type ListNotesFilter struct {
	FolderID string
	Query    string
	Status   string
	Limit    int
	Offset   int
}

type CreateNoteInput struct {
	Title     string `json:"title"`
	FolderID  *uint  `json:"folder_id"`
	ContentMD string `json:"content_md"`
}

type UpdateNoteInput struct {
	Title       *string      `json:"title"`
	FolderID    OptionalUint `json:"folder_id"`
	CoverURL    *string      `json:"cover_url"`
	ContentMD   *string      `json:"content_md"`
	IsPublished *bool        `json:"is_published"`
	Visibility  *string      `json:"visibility"`
	UpdatedAt   string       `json:"updated_at"`
}

type Folder struct {
	ID        uint   `json:"id"`
	UserID    uint   `json:"user_id"`
	Name      string `json:"name"`
	ParentID  *uint  `json:"parent_id"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type FolderList struct {
	Items []Folder `json:"items"`
}

type CreateFolderInput struct {
	Name     string `json:"name" binding:"required"`
	ParentID *uint  `json:"parent_id"`
}

type UpdateFolderInput struct {
	Name     string       `json:"name"`
	ParentID OptionalUint `json:"parent_id"`
}

type ReorderInput struct {
	Folders []ReorderFolder `json:"folders"`
	Notes   []ReorderNote   `json:"notes"`
}

type ReorderFolder struct {
	ID        uint `json:"id"`
	SortOrder int  `json:"sort_order"`
}

type ReorderNote struct {
	ID        uint  `json:"id"`
	SortOrder int   `json:"sort_order"`
	FolderID  *uint `json:"folder_id"`
}

type PublicNoteListItem struct {
	ID             uint   `json:"id"`
	Title          string `json:"title"`
	Slug           string `json:"slug"`
	UserID         uint   `json:"user_id"`
	AuthorUsername string `json:"author_username"`
	Excerpt        string `json:"excerpt"`
	CoverURL       string `json:"cover_url"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type PublicNotePage struct {
	Items  []PublicNoteListItem `json:"items"`
	Total  int64                `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

type PublicNote struct {
	Note
	AuthorUsername string `json:"author_username"`
}

func formatTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func nowTimestamp() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}
