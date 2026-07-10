package repository

import (
	"errors"
	"financial/system/api/config"
	"financial/system/api/entities"

	"gorm.io/gorm"
)

func CreateAccountRepository(name, email, cpf string) ([]map[string]interface{}, error) {
	db := config.GetPostgres()
	logger := config.GetLogger()

	var existingAccounts []entities.Account
	var newAccount entities.Account

	err := db.Transaction(func(tx *gorm.DB) error {
		var client entities.Client

		logger.Infof("Searching client with CPF (%s) and Email (%s)", cpf, email)
		err := tx.Where("cpf = ? OR email = ?", cpf, email).First(&client).Error

		if err == nil {
			if client.CPF != cpf {
				logger.Errorf("CPF mismatch for client with email %s: expected %s, got %s", email, client.CPF, cpf)
				return errors.New("This email is already in use by another user.")
			}

			logger.Infof("Client found (ID: %d). Searching for existing accounts...", client.ID)

			if errAccounts := tx.Where("client_id = ?", client.ID).Find(&existingAccounts).Error; errAccounts != nil {
				logger.Errorf("Error searching for existing accounts: %v", errAccounts)
				return errAccounts
			}

		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Infof("Client not found, creating new client with cpf: %s", cpf)

			client = entities.Client{
				Name:  name,
				Email: email,
				CPF:   cpf,
			}

			if errCreate := tx.Create(&client).Error; errCreate != nil {
				logger.Errorf("Error creating client: %v", errCreate)
				return errCreate
			}
		} else {
			logger.Errorf("Error occurred while fetching client on database: %v", err)
			return err
		}

		logger.Infof("Creating account with ClienteID: %d", client.ID)
		newAccount = entities.Account{
			ClientID:         client.ID,
			AvailableBalance: 0,
			ReservedBalance:  0,
			CreditLimit:      5000,
			Currency:         "BRL",
			Status:           "active",
		}

		if errCreateAcc := tx.Create(&newAccount).Error; errCreateAcc != nil {
			logger.Errorf("Error creating account for client %d: %v", client.ID, errCreateAcc)
			return errCreateAcc
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	var response []map[string]interface{}
	for _, acc := range existingAccounts {
		response = append(response, map[string]interface{}{
			"account_id": acc.ID,
			"client_id":  acc.ClientID,
			"status":     acc.Status,
			"currency":   acc.Currency,
			"type":       "existing",
		})
	}

	response = append(response, map[string]interface{}{
		"account_id": newAccount.ID,
		"client_id":  newAccount.ClientID,
		"status":     newAccount.Status,
		"currency":   newAccount.Currency,
		"type":       "new",
	})

	return response, nil
}
