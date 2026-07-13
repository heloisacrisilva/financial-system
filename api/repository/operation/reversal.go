package repository

import (
	"errors"
	"financial/system/api/config"
	"financial/system/api/entities"
	"financial/system/api/helpers"
	repositoryErrors "financial/system/api/repository"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const TransactionTypeReversal = "reversal"

func ReversalOperation(originalRefID string, refID string, requestedValue int64) ([]*entities.Account, []entities.TransactionHistory, error) {
	db := config.GetPostgres()
	logger := config.GetLogger()

	var accounts []*entities.Account
	var histories []entities.TransactionHistory

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := reserveReferenceID(tx, refID); err != nil {
			return err
		}

		var originals []entities.TransactionHistory
		if err := tx.Where("reference_id = ?", originalRefID).Find(&originals).Error; err != nil {
			return err
		}
		if len(originals) == 0 {
			return repositoryErrors.ErrTransactionNotFound
		}

		origType := originals[0].Type
		if origType != "credit" && origType != "debit" && origType != "transfer" {
			return repositoryErrors.ErrTransactionNotReversible
		}

		for _, o := range originals {
			if o.Status != "success" {
				return repositoryErrors.ErrTransactionNotReversible
			}
		}

		ids := make([]uint64, len(originals))
		for i, o := range originals {
			ids[i] = o.ID
		}
		var alreadyReversedCount int64
		if err := tx.Model(&entities.TransactionHistory{}).
			Where("type = ? AND related_tx_id IN ?", TransactionTypeReversal, ids).
			Count(&alreadyReversedCount).Error; err != nil {
			return err
		}
		if alreadyReversedCount > 0 {
			return repositoryErrors.ErrAlreadyReversed
		}

		switch origType {
		case "credit", "debit":
			return reverseSimpleOperation(tx, originals[0], refID, &accounts, &histories)
		case "transfer":
			if len(originals) != 2 {
				return repositoryErrors.ErrTransactionNotReversible
			}
			return reverseTransferOperation(tx, originals, refID, &accounts, &histories)
		}

		return repositoryErrors.ErrTransactionNotReversible
	})

	if err != nil {
		logger.Errorf("reversal failed original_ref=%s ref=%s: %v", originalRefID, refID, err)
		return nil, nil, err
	}

	return accounts, histories, nil
}

func reverseSimpleOperation(tx *gorm.DB, original entities.TransactionHistory, refID string, accounts *[]*entities.Account, histories *[]entities.TransactionHistory) error {
	var account entities.Account
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&account, "id = ?", original.AccountID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repositoryErrors.ErrAccountNotFound
		}
		return err
	}

	switch original.Type {
	case "credit":
		if account.AvailableBalance < original.Value {
			return repositoryErrors.ErrInsufficientFunds
		}
		account.AvailableBalance -= original.Value
	case "debit":
		account.AvailableBalance += original.Value
	}

	if err := tx.Save(&account).Error; err != nil {
		return err
	}

	hist := entities.TransactionHistory{
		AccountID:   account.ID,
		ReferenceID: refID,
		Type:        TransactionTypeReversal,
		Value:       original.Value,
		Currency:    original.Currency,
		Status:      "success",
		RelatedTxID: &original.ID,
		CreatedAt:   time.Now(),
	}
	if err := tx.Create(&hist).Error; err != nil {
		if helpers.IsUniqueViolation(err) {
			return repositoryErrors.ErrDuplicateRef
		}
		return err
	}
	if err := enqueueOperationEvent(tx, "operation.reversal.success", refID, map[string]interface{}{
		"transaction_id":          hist.ID,
		"original_transaction_id": original.ID,
		"account_id":              account.ID,
		"value":                   original.Value,
		"currency":                original.Currency,
		"status":                  hist.Status,
	}); err != nil {
		return err
	}

	*accounts = append(*accounts, &account)
	*histories = append(*histories, hist)
	return nil
}

func reverseTransferOperation(tx *gorm.DB, originals []entities.TransactionHistory, refID string, accounts *[]*entities.Account, histories *[]entities.TransactionHistory) error {
	var debitLeg, creditLeg entities.TransactionHistory
	for _, o := range originals {
		if o.Direction == nil {
			return repositoryErrors.ErrTransactionNotReversible
		}
		switch *o.Direction {
		case "debit":
			debitLeg = o
		case "credit":
			creditLeg = o
		}
	}
	if debitLeg.ID == 0 || creditLeg.ID == 0 {
		return repositoryErrors.ErrTransactionNotReversible
	}

	origOriginID := debitLeg.AccountID
	origDestID := creditLeg.AccountID

	firstID, secondID := origOriginID, origDestID
	if secondID < firstID {
		firstID, secondID = secondID, firstID
	}

	var first, second entities.Account
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&first, "id = ?", firstID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repositoryErrors.ErrAccountNotFound
		}
		return err
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&second, "id = ?", secondID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repositoryErrors.ErrAccountNotFound
		}
		return err
	}

	var origOrigin, origDest *entities.Account
	if firstID == origOriginID {
		origOrigin, origDest = &first, &second
	} else {
		origOrigin, origDest = &second, &first
	}

	if origDest.AvailableBalance < debitLeg.Value {
		return repositoryErrors.ErrInsufficientFunds
	}

	origDest.AvailableBalance -= debitLeg.Value
	origOrigin.AvailableBalance += debitLeg.Value

	if err := tx.Save(&origOrigin).Error; err != nil {
		return err
	}
	if err := tx.Save(&origDest).Error; err != nil {
		return err
	}

	transferGroupID := uuid.NewString()

	creditRev := entities.TransactionHistory{
		AccountID:       origOrigin.ID,
		ReferenceID:     refID,
		Type:            TransactionTypeReversal,
		Direction:       helpers.StrPtr("credit"),
		Value:           debitLeg.Value,
		Currency:        debitLeg.Currency,
		Status:          "success",
		TransferGroupID: &transferGroupID,
		RelatedTxID:     &debitLeg.ID,
		CreatedAt:       time.Now(),
	}
	if err := tx.Create(&creditRev).Error; err != nil {
		if helpers.IsUniqueViolation(err) {
			return repositoryErrors.ErrDuplicateRef
		}
		return err
	}

	debitRev := entities.TransactionHistory{
		AccountID:       origDest.ID,
		ReferenceID:     refID,
		Type:            TransactionTypeReversal,
		Direction:       helpers.StrPtr("debit"),
		Value:           debitLeg.Value,
		Currency:        creditLeg.Currency,
		Status:          "success",
		TransferGroupID: &transferGroupID,
		RelatedTxID:     &creditLeg.ID,
		CreatedAt:       time.Now(),
	}
	if err := tx.Create(&debitRev).Error; err != nil {
		if helpers.IsUniqueViolation(err) {
			return repositoryErrors.ErrDuplicateRef
		}
		return err
	}
	if err := enqueueOperationEvent(tx, "operation.reversal.success", refID, map[string]interface{}{
		"debit_transaction_id":  debitRev.ID,
		"credit_transaction_id": creditRev.ID,
		"original_debit_tx_id":  debitLeg.ID,
		"original_credit_tx_id": creditLeg.ID,
		"origin_account_id":     origOrigin.ID,
		"dest_account_id":       origDest.ID,
		"value":                 debitLeg.Value,
		"currency":              debitLeg.Currency,
		"transfer_group_id":     transferGroupID,
		"status":                debitRev.Status,
	}); err != nil {
		return err
	}

	*accounts = append(*accounts, origOrigin, origDest)
	*histories = append(*histories, debitRev, creditRev)
	return nil
}
