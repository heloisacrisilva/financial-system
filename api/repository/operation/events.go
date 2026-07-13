package repository

import (
	"encoding/json"
	"financial/system/api/entities"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func enqueueOperationEvent(tx *gorm.DB, eventType string, refID string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	now := time.Now()
	event := entities.OperationEvent{
		ReferenceID:   refID,
		EventType:     eventType,
		Payload:       datatypes.JSON(payloadJSON),
		Status:        "pending",
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	return tx.Create(&event).Error
}
