package repository

import (
	"financial/system/api/config"
	"financial/system/api/entities"
)

func GetAccountRepository(accountID string) (entities.Account, error) {
	db := config.GetPostgres()
	var accountData entities.Account

	if err := db.Preload("Client").First(&accountData, "id = ?", accountID).Error; err != nil {
		return entities.Account{}, err
	}

	return accountData, nil
}
