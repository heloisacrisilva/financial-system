package repostiory

import (
	"errors"
	"financial/system/api/config"
	"financial/system/api/entities"
	"financial/system/api/helpers"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInsufficientFunds = errors.New("insuficient available_balance")
)

func DebitOperation(accountID string, currency string, value int64, refID string) (*entities.Account, *entities.TransactionHistory, error) {
	db := config.GetPostgres()
	logger := config.GetLogger()

	var account entities.Account
	var history entities.TransactionHistory

	err := db.Transaction(func(tx *gorm.DB) error {
		var existing entities.TransactionHistory
		err := tx.Where("reference_id = ?", refID).First(&existing).Error

		if err == nil {
			return ErrDuplicateRef
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, "id = ?", accountID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAccountNotFound
			}
			return err
		}

		if account.Status != "active" {
			return ErrAccountNotActive
		}

		if currency != "" && currency != account.Currency {
			return ErrInvalidCurrency
		}

		totalAvailable := account.AvailableBalance + account.CreditLimit
		if totalAvailable < value {
			return ErrInsufficientFunds
		}

		account.AvailableBalance -= value
		account.Version++

		if err := tx.Save(&account).Error; err != nil {
			return err
		}

		history = entities.TransactionHistory{
			AccountID:   accountID,
			ReferenceID: refID,
			Type:        "debit",
			Value:       value,
			Currency:    account.Currency,
			Status:      "success",
			CreatedAt:   time.Now(),
		}

		if err := tx.Create(&history).Error; err != nil {
			if helpers.IsUniqueViolation(err) {
				return ErrDuplicateRef
			}
			return err
		}
		return nil
	})

	if err != nil {
		logger.Errorf("debit failed account=%s ref=%s: %v", accountID, refID, err)
		return nil, nil, err
	}

	return &account, &history, nil
}

func RecordFailedDedit(accountID, refID string, value int64, currency string, cause error) {
	db := config.GetPostgres()
	logger := config.GetLogger()
	errMsg := cause.Error()

	h := entities.TransactionHistory{
		AccountID:    accountID,
		ReferenceID:  refID,
		Type:         "debit",
		Value:        value,
		Currency:     currency,
		Status:       "failed",
		ErrorMessage: &errMsg,
		CreatedAt:    time.Now(),
	}
	if err := db.Create(&h).Error; err != nil && !helpers.IsUniqueViolation(err) {
		logger.Errorf("failed to record failed debit history ref=%s: %v", refID, err)
	}
}
