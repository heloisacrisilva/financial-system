package entities

import (
	"time"

	"gorm.io/datatypes"
)

type OperationEvent struct {
	ID            uint64         `gorm:"primaryKey;autoIncrement"`
	ReferenceID   string         `gorm:"type:varchar(100);not null;index"`
	EventType     string         `gorm:"type:varchar(100);not null;index"`
	Payload       datatypes.JSON `gorm:"not null"`
	Status        string         `gorm:"not null;index;check:status IN ('pending','published','failed')"`
	Attempts      int            `gorm:"not null;default:0"`
	LastError     *string        `gorm:"type:text"`
	NextAttemptAt time.Time      `gorm:"not null;index"`
	PublishedAt   *time.Time
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}
