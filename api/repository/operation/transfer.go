package repository

import (
	"errors"
	"financial/system/api/config"
	"financial/system/api/entities"
	"financial/system/api/helpers"
	"financial/system/api/repository"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TransferOperation(accountOrigID uint64, accountDestID uint64, currency string, value int64, refID string) ([]entities.TransactionHistory, *entities.Account, *entities.Account, error) {
	db := config.GetPostgres()
	logger := config.GetLogger()

	var originAccount, destAccount entities.Account
	var debitHistory, creditHistory entities.TransactionHistory
	const TransactionTypeTransfer = "transfer"

	err := db.Transaction(func(tx *gorm.DB) error {
		if accountOrigID == accountDestID {
			return repository.ErrSameAccount
		}

		if err := reserveReferenceID(tx, refID); err != nil {
			return err
		}

		firstID, secondID := accountOrigID, accountDestID
		if secondID < firstID {
			firstID, secondID = secondID, firstID
		}

		var first, second entities.Account

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&first, "id = ?", firstID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrAccountNotFound
			}
			return err
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&second, "id = ?", secondID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrAccountNotFound
			}
			return err
		}

		if firstID == accountOrigID {
			originAccount, destAccount = first, second
		} else {
			originAccount, destAccount = second, first
		}

		if err := validateAccountForTransfer(originAccount, currency); err != nil {
			return err
		}
		if err := validateAccountForTransfer(destAccount, currency); err != nil {
			return err
		}

		if originAccount.Currency != destAccount.Currency {
			return repository.ErrInvalidCurrency
		}

		if originAccount.AvailableBalance < value {
			return repository.ErrInsufficientFunds
		}

		originAccount.AvailableBalance -= value
		destAccount.AvailableBalance += value

		if err := tx.Save(&originAccount).Error; err != nil {
			return err
		}
		if err := tx.Save(&destAccount).Error; err != nil {
			return err
		}

		transferGroupID := uuid.NewString()

		debitHistory = entities.TransactionHistory{
			AccountID:       accountOrigID,
			ReferenceID:     refID,
			Type:            TransactionTypeTransfer,
			Direction:       helpers.StrPtr("debit"),
			Value:           value,
			Currency:        originAccount.Currency,
			Status:          "success",
			TransferGroupID: &transferGroupID,
			CreatedAt:       time.Now(),
		}
		if err := tx.Create(&debitHistory).Error; err != nil {
			if helpers.IsUniqueViolation(err) {
				return repository.ErrDuplicateRef
			}
			return err
		}

		creditHistory = entities.TransactionHistory{
			AccountID:       accountDestID,
			ReferenceID:     refID,
			Type:            TransactionTypeTransfer,
			Direction:       helpers.StrPtr("credit"),
			Value:           value,
			Currency:        destAccount.Currency,
			Status:          "success",
			TransferGroupID: &transferGroupID,
			RelatedTxID:     &debitHistory.ID,
			CreatedAt:       time.Now(),
		}
		if err := tx.Create(&creditHistory).Error; err != nil {
			if helpers.IsUniqueViolation(err) {
				return repository.ErrDuplicateRef
			}
			return err
		}

		debitHistory.RelatedTxID = &creditHistory.ID
		if err := tx.Save(&debitHistory).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		logger.Errorf("%s failed orig=%d dest=%d ref=%s: %v",
			TransactionTypeTransfer, accountOrigID, accountDestID, refID, err)
		return nil, nil, nil, err
	}

	return []entities.TransactionHistory{debitHistory, creditHistory}, &originAccount, &destAccount, nil
}

func validateAccountForTransfer(acc entities.Account, currency string) error {
	if acc.Status != "active" {
		return repository.ErrAccountNotActive
	}
	if currency != "" && currency != acc.Currency {
		return repository.ErrInvalidCurrency
	}
	return nil
}
