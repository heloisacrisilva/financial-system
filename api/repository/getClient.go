package repository

import (
	"financial/system/api/config"
	"financial/system/api/entities"
)

func GetClientRepository(clientID string) (entities.Client, error) {
	db := config.GetPostgres()
	var clientData entities.Client

	if err := db.Preload("Accounts").First(&clientData, "id = ?", clientID).Error; err != nil {
		return entities.Client{}, err
	}
	return clientData, nil
}
