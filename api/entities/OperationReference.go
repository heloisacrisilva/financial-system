package entities

import "time"

type OperationReference struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	ReferenceID string    `gorm:"type:varchar(100);not null;uniqueIndex"`
	CreatedAt   time.Time `gorm:"not null"`
}
