package entities

import (
	"time"

	"gorm.io/datatypes"
)

type TransactionHistory struct {
	ID              uint64  `gorm:"primaryKey;autoIncrement"`
	AccountID       uint64  `gorm:"not null;index"`
	Account         Account `gorm:"foreignKey:AccountID;references:ID;OnDelete:RESTRICT"`
	ReferenceID     string  `gorm:"type:varchar(100);not null;index"`
	TransferGroupID *string `gorm:"index"`
	RelatedTxID     *uint64 `gorm:"index"`
	Type            string  `gorm:"not null;check:type IN ('credit', 'debit', 'reserve', 'capture', 'reversal', 'transfer')"`
	Direction       *string `gorm:"type:varchar(10);check:direction IN ('debit','credit')"`
	Value           int64   `gorm:"not null;"`
	Currency        string  `gorm:"type:varchar(3);not null"`
	Status          string  `gorm:"not null;check:status IN ('success','failed','pending')"`
	ErrorMessage    *string `gorm:"type:text"`
	Metadata        datatypes.JSON
	CreatedAt       time.Time `gorm:"not null"`
}
