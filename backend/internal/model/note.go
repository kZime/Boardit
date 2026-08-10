// Model for note in the database
package model

import "time"

type Note struct {
	ID          uint      `gorm:"primaryKey"                                          json:"id"`
	UserID      uint      `gorm:"not null;uniqueIndex:idx_user_slug;constraint:OnUpdate:CASCADE" json:"user_id"`
	FolderID    *uint     `gorm:"index;constraint:OnUpdate:CASCADE"                   json:"folder_id"`
	Title       string    `gorm:"type:varchar(255);not null"                          json:"title"`
	Slug        string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_user_slug" json:"slug"`
	CoverURL    string    `gorm:"type:varchar(2048);not null;default:''"              json:"cover_url"`
	ContentMd   string    `gorm:"type:text;not null"                                  json:"content_md"`
	ContentHtml string    `gorm:"type:text;not null"                                  json:"content_html"`
	IsPublished bool      `gorm:"not null;default:false"                              json:"is_published"`
	Visibility  string    `gorm:"type:varchar(20);not null;default:'private'"         json:"visibility"`
	SortOrder   int       `gorm:"not null;default:0"                                  json:"sort_order"`
	Version     uint64    `gorm:"not null;default:1"                                  json:"version"`
	CreatedAt   time.Time `gorm:"not null"                                            json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null"                                            json:"updated_at"`
}
