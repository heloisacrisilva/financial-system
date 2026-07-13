package repository

import (
	"financial/system/api/entities"
	"financial/system/api/helpers"
	repositoryErrors "financial/system/api/repository"
	"time"

	"gorm.io/gorm"
)

func reserveReferenceID(tx *gorm.DB, refID string) error {
	ref := entities.OperationReference{
		ReferenceID: refID,
		CreatedAt:   time.Now(),
	}

	if err := tx.Create(&ref).Error; err != nil {
		if helpers.IsUniqueViolation(err) {
			return repositoryErrors.ErrDuplicateRef
		}
		return err
	}

	return nil
}
