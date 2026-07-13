package events

import (
	"context"
	"errors"
	"financial/system/api/config"
	"financial/system/api/entities"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxAttempts  = 5
	pollInterval = 2 * time.Second
	batchSize    = 20
)

func StartDispatcher(ctx context.Context) {
	logger := config.GetLogger()

	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		logger.Info("Operation event dispatcher started")

		for {
			processPendingEvents(ctx)

			select {
			case <-ctx.Done():
				logger.Info("Operation event dispatcher stopped")
				return
			case <-ticker.C:
			}
		}
	}()
}

func processPendingEvents(ctx context.Context) {
	db := config.GetPostgres()
	logger := config.GetLogger()

	if db == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var events []entities.OperationEvent
		err := db.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status IN ? AND next_attempt_at <= ?", []string{"pending", "failed"}, time.Now()).
			Order("created_at ASC").
			Limit(batchSize).
			Find(&events).Error
		if err != nil {
			logger.Errorf("failed to fetch operation events: %v", err)
			return
		}
		if len(events) == 0 {
			return
		}

		for _, event := range events {
			if err := publishEvent(ctx, event); err != nil {
				logger.Errorf("failed to publish operation event id=%d ref=%s type=%s: %v", event.ID, event.ReferenceID, event.EventType, err)
				markEventFailed(db, event, err)
				continue
			}

			now := time.Now()
			if err := db.Model(&entities.OperationEvent{}).
				Where("id = ?", event.ID).
				Updates(map[string]interface{}{
					"status":       "published",
					"published_at": &now,
					"updated_at":   now,
					"last_error":   nil,
				}).Error; err != nil {
				logger.Errorf("failed to mark operation event id=%d as published: %v", event.ID, err)
			}
		}
	}
}

func publishEvent(ctx context.Context, event entities.OperationEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if event.EventType == "" {
		return errors.New("event type is empty")
	}

	config.GetLogger().Infof("published operation event id=%d ref=%s type=%s payload=%s", event.ID, event.ReferenceID, event.EventType, string(event.Payload))
	return nil
}

func markEventFailed(db *gorm.DB, event entities.OperationEvent, cause error) {
	now := time.Now()
	attempts := event.Attempts + 1
	status := "failed"
	nextAttemptAt := now.Add(backoff(attempts))
	errMsg := cause.Error()

	if attempts >= maxAttempts {
		nextAttemptAt = now.Add(24 * time.Hour)
	}

	_ = db.Model(&entities.OperationEvent{}).
		Where("id = ?", event.ID).
		Updates(map[string]interface{}{
			"status":          status,
			"attempts":        attempts,
			"last_error":      &errMsg,
			"next_attempt_at": nextAttemptAt,
			"updated_at":      now,
		}).Error
}

func backoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	if attempts > maxAttempts {
		attempts = maxAttempts
	}
	return time.Duration(1<<uint(attempts-1)) * time.Second
}
