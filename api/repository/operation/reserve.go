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

func ReserveOperation(accountID uint64, currency string, value int64, refID string) (*entities.Account, *entities.TransactionHistory, error) {
	db := config.GetPostgres()
	logger := config.GetLogger()

	var account entities.Account
	var history entities.TransactionHistory
	const TransactionTypeReserve = "reserve"

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := reserveReferenceID(tx, refID); err != nil {
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

		if value <= 0 {
			return repository.ErrInvalidValue
		}

		if value > account.AvailableBalance {
			return repository.ErrInsufficientFunds
		}

		account.AvailableBalance -= value
		account.ReservedBalance += value

		if err := tx.Save(&account).Error; err != nil {
			return err
		}

		history = entities.TransactionHistory{
			AccountID:   accountID,
			ReferenceID: refID,
			Type:        TransactionTypeReserve,
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
		if err := enqueueOperationEvent(tx, "operation.reserve.success", refID, map[string]interface{}{
			"transaction_id":    history.ID,
			"account_id":        account.ID,
			"value":             value,
			"currency":          account.Currency,
			"available_balance": account.AvailableBalance,
			"reserved_balance":  account.ReservedBalance,
			"status":            history.Status,
		}); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		logger.Errorf("%s failed account=%d ref=%s: %v", TransactionTypeReserve, accountID, refID, err)
		return nil, nil, err
	}

	return &account, &history, nil
}
