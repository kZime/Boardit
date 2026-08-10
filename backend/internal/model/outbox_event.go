package model

import "time"

// OutboxEvent is written in the same transaction as the aggregate change.
// A future worker may claim pending rows without coupling note use-cases to a queue vendor.
type OutboxEvent struct {
	ID               uint      `gorm:"primaryKey"`
	UserID           uint      `gorm:"not null"`
	AggregateType    string    `gorm:"type:varchar(80);not null"`
	AggregateID      uint      `gorm:"not null"`
	AggregateVersion uint64    `gorm:"not null"`
	EventType        string    `gorm:"type:varchar(120);not null"`
	Payload          string    `gorm:"not null"`
	Status           string    `gorm:"type:varchar(20);not null;default:'pending'"`
	AvailableAt      time.Time `gorm:"not null"`
	CreatedAt        time.Time `gorm:"not null"`
	ProcessedAt      *time.Time
}
