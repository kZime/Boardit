package model

import "time"

type RefreshSession struct {
	ID                uint      `gorm:"primaryKey"`
	UserID            uint      `gorm:"not null;index"`
	TokenID           string    `gorm:"type:varchar(64);not null;uniqueIndex"`
	FamilyID          string    `gorm:"type:varchar(64);not null;index"`
	ExpiresAt         time.Time `gorm:"not null"`
	RevokedAt         *time.Time
	ReplacedByTokenID *string `gorm:"type:varchar(64)"`
	CreatedAt         time.Time
}
