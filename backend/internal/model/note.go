// Model for note in the database
package model

import "time"

type Note struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uint      `gorm:"not null;index;constraint:OnUpdate:CASCADE"`
	FolderID     *uint     `gorm:"index;constraint:OnUpdate:CASCADE"`
	Title        string    `gorm:"type:varchar(255);not null"`
	Slug         string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_user_slug"`
	ContentMd    string    `gorm:"type:text;not null"`
	ContentHtml  string    `gorm:"type:text;not null"`
	IsPublished  bool      `gorm:"not null;default:false"`
	Visibility   string    `gorm:"type:varchar(20);not null;default:'private'"`
	SortOrder    int       `gorm:"not null;default:0"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}