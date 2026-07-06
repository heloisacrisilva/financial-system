package entities

import (
	"time"
)

type Client struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"type:varchar(255);not null;"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	CPF       string    `gorm:"type:varchar(11);not null;unique;"`
	Status    string    `gorm:"not null;default:'active';check:status IN ('active','closed')"`
	Accounts  []Account `gorm:"foreignKey:ClientID;references:ID;constraint:OnDelete:RESTRICT"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}
