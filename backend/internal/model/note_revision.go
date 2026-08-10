// Model for note revision in the database
package model

import "time"

type NoteRevision struct {
	ID          uint      `gorm:"primaryKey"                                       json:"id"`
	NoteID      uint      `gorm:"not null;uniqueIndex:idx_note_revision_version;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"note_id"`
	UserID      uint      `gorm:"not null;index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user_id"`
	Version     uint64    `gorm:"not null;uniqueIndex:idx_note_revision_version"    json:"version"`
	Title       string    `gorm:"type:varchar(255);not null"                       json:"title"`
	ContentMd   string    `gorm:"type:text;not null"                               json:"content_md"`
	ContentHtml string    `gorm:"type:text;not null"                               json:"content_html"`
	Diff        *string   `gorm:"type:text"                                        json:"diff"`
	Source      string    `gorm:"type:varchar(40);not null;default:'user'"          json:"source"`
	CreatedAt   time.Time `gorm:"not null"                                         json:"created_at"`
}
