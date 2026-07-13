package repository

import (
	"financial/system/api/config"
	"financial/system/api/entities"
	"financial/system/api/helpers"
	"time"
)

func RecordFailedOperation(typeOpt string, accountID uint64, refID string, value int64, currency string, cause error) {
	db := config.GetPostgres()
	logger := config.GetLogger()
	errMsg := cause.Error()

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
	if err := db.Create(&h).Error; err != nil && !helpers.IsUniqueViolation(err) {
		logger.Errorf("failed to record failed %s history ref=%s: %v", typeOpt, refID, err)
	}
}
