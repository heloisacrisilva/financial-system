package entities

import (
	"time"
)

type Account struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement"`
	ClientID         uint64    `gorm:"not null;index"`
	Client           Client    `gorm:"foreignKey:ClientID;references:ID" json:"-"`
	AvailableBalance int64     `gorm:"not null"`
	ReservedBalance  int64     `gorm:"not null"`
	CreditLimit      int64     `gorm:"not null"`
	Currency         string    `gorm:"type:varchar(3);not null;default:'BRL'"`
	Status           string    `gorm:"not null;check:status IN ('active','inactive','blocked')"`
	Version          uint      `gorm:"not null;default:0"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
}
