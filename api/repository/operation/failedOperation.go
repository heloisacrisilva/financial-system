package repository

import (
	"errors"
	"financial/system/api/config"
	"financial/system/api/entities"
	"financial/system/api/helpers"
	repositoryErrors "financial/system/api/repository"
	"time"

	"gorm.io/gorm"
)

func RecordFailedOperation(typeOpt string, accountID uint64, refID string, value int64, currency string, cause error) {
	db := config.GetPostgres()
	logger := config.GetLogger()
	errMsg := cause.Error()

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := reserveReferenceID(tx, refID); err != nil {
			return err
		}

		h := entities.TransactionHistory{
			AccountID:    accountID,
			ReferenceID:  refID,
			Type:         typeOpt,
			Value:        value,
			Currency:     currency,
			Status:       "failed",
			ErrorMessage: &errMsg,
			CreatedAt:    time.Now(),
		}
		if err := tx.Create(&h).Error; err != nil {
			return err
		}

		return enqueueOperationEvent(tx, "operation."+typeOpt+".failed", refID, map[string]interface{}{
			"transaction_id": h.ID,
			"account_id":     accountID,
			"value":          value,
			"currency":       currency,
			"status":         h.Status,
			"error_message":  errMsg,
		})
	})
	if err != nil && !errors.Is(err, repositoryErrors.ErrDuplicateRef) && !helpers.IsUniqueViolation(err) {
		logger.Errorf("failed to record failed %s history ref=%s: %v", typeOpt, refID, err)
	}
}
