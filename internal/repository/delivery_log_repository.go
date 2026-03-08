package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/realtime-notification-system/notification-service/internal/models"
)

type DeliveryLogRepository interface {
	Create(ctx context.Context, log *models.NotificationDeliveryLog) error
	GetByNotificationID(ctx context.Context, notificationID uuid.UUID) ([]*models.NotificationDeliveryLog, error)
}

type deliveryLogRepository struct {
	db *sql.DB
}

func NewDeliveryLogRepository(db *sql.DB) DeliveryLogRepository {
	return &deliveryLogRepository{db: db}
}

func (r *deliveryLogRepository) Create(ctx context.Context, log *models.NotificationDeliveryLog) error {
	query := `
		INSERT INTO notification_delivery_logs (notification_id, attempt_number, status, error_message, processing_time_ms)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	err := r.db.QueryRowContext(ctx, query,
		log.NotificationID, log.AttemptNumber, log.Status, log.ErrorMessage, log.ProcessingTimeMs,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create delivery log: %w", err)
	}

	return nil
}

func (r *deliveryLogRepository) GetByNotificationID(ctx context.Context, notificationID uuid.UUID) ([]*models.NotificationDeliveryLog, error) {
	query := `
		SELECT id, notification_id, attempt_number, status, error_message, processing_time_ms, created_at
		FROM notification_delivery_logs
		WHERE notification_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query delivery logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.NotificationDeliveryLog
	for rows.Next() {
		log := &models.NotificationDeliveryLog{}
		err := rows.Scan(
			&log.ID, &log.NotificationID, &log.AttemptNumber, &log.Status,
			&log.ErrorMessage, &log.ProcessingTimeMs, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delivery log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}
