// Model for folder in the database
package model

import "time"

type Folder struct {
	ID        uint      `gorm:"primaryKey"                                 json:"id"`
	UserID    uint      `gorm:"not null;index;constraint:OnUpdate:CASCADE" json:"user_id"`
	Name      string    `gorm:"type:varchar(255);not null"                 json:"name"`
	ParentID  *uint     `gorm:"index;constraint:OnUpdate:CASCADE"          json:"parent_id"`
	SortOrder int       `gorm:"not null;default:0"                         json:"sort_order"`
	CreatedAt time.Time `gorm:"not null"                                   json:"created_at"`
	UpdatedAt time.Time `gorm:"not null"                                   json:"updated_at"`
}
