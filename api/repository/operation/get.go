package repository

import (
	"financial/system/api/config"
	"financial/system/api/entities"
)

func GetOperationHistoryByID(operationID uint64) (entities.TransactionHistory, error) {
	db := config.GetPostgres()
	var history entities.TransactionHistory

	if err := db.First(&history, "id = ?", operationID).Error; err != nil {
		return entities.TransactionHistory{}, err
	}

	return history, nil
}
