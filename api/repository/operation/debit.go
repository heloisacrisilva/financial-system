package repository

import (
	"errors"
	"financial/system/api/config"
	"financial/system/api/entities"
	"financial/system/api/helpers"
	"financial/system/api/repository"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func DebitOperation(accountID uint64, currency string, value int64, refID string) (*entities.Account, *entities.TransactionHistory, error) {
	db := config.GetPostgres()
	logger := config.GetLogger()

	var account entities.Account
	var history entities.TransactionHistory
	const TransactionTypeDebit = "debit"

	err := db.Transaction(func(tx *gorm.DB) error {
		var existing entities.TransactionHistory
		err := tx.Where("reference_id = ? AND type = ?", refID, TransactionTypeDebit).First(&existing).Error

		if err == nil {
			return repository.ErrDuplicateRef
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, "id = ?", accountID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrAccountNotFound
			}
			return err
		}

		if account.Status != "active" {
			return repository.ErrAccountNotActive
		}

		if currency != "" && currency != account.Currency {
			return repository.ErrInvalidCurrency
		}

		totalAvailable := account.AvailableBalance + account.CreditLimit
		if totalAvailable < value {
			return repository.ErrInsufficientFunds
		}

		account.AvailableBalance -= value
		account.Version++

		if err := tx.Save(&account).Error; err != nil {
			return err
		}

		history = entities.TransactionHistory{
			AccountID:   accountID,
			ReferenceID: refID,
			Type:        TransactionTypeDebit,
			Value:       value,
			Currency:    account.Currency,
			Status:      "success",
			CreatedAt:   time.Now(),
		}

		if err := tx.Create(&history).Error; err != nil {
			if helpers.IsUniqueViolation(err) {
				return repository.ErrDuplicateRef
			}
			return err
		}
		return nil
	})

	if err != nil {
		logger.Errorf("%s failed account=%d ref=%s: %v", TransactionTypeDebit, accountID, refID, err)
		return nil, nil, err
	}

	return &account, &history, nil
}
