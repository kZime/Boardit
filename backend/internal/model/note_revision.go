// Model for note revision in the database
package model

import "time"

type NoteRevision struct {
	ID        uint      `gorm:"primaryKey"`
	NoteID    uint      `gorm:"not null;index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ContentMd string    `gorm:"type:text;not null"`
	Diff      *string   `gorm:"type:text"`
	CreatedAt time.Time `gorm:"not null"`
}
