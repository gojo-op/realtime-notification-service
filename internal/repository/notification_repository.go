package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/realtime-notification-system/notification-service/internal/models"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	CreateBatch(ctx context.Context, notifications []*models.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int, filters map[string]interface{}) ([]*models.Notification, int, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.NotificationStatus) error
	UpdateDeliveryTime(ctx context.Context, id uuid.UUID, deliveredAt time.Time) error
}

type notificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	metadataJSON, err := json.Marshal(notification.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO notifications (user_id, title, message, type, priority, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, status, is_read, created_at, sent_at, delivered_at`

	err = r.db.QueryRowContext(ctx, query,
		notification.UserID, notification.Title, notification.Message,
		notification.Type, notification.Priority, metadataJSON,
	).Scan(&notification.ID, &notification.Status, &notification.IsRead,
		&notification.CreatedAt, &notification.SentAt, &notification.DeliveredAt)

	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

func (r *notificationRepository) CreateBatch(ctx context.Context, notifications []*models.Notification) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO notifications (user_id, title, message, type, priority, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, status, is_read, created_at, sent_at, delivered_at`

	for _, notification := range notifications {
		metadataJSON, err := json.Marshal(notification.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		err = tx.QueryRowContext(ctx, query,
			notification.UserID, notification.Title, notification.Message,
			notification.Type, notification.Priority, metadataJSON,
		).Scan(&notification.ID, &notification.Status, &notification.IsRead,
			&notification.CreatedAt, &notification.SentAt, &notification.DeliveredAt)

		if err != nil {
			return fmt.Errorf("failed to create notification: %w", err)
		}
	}

	return tx.Commit()
}

func (r *notificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	notification := &models.Notification{}
	var metadataJSON []byte

	query := `
		SELECT id, user_id, title, message, type, priority, status, is_read, read_at,
		       metadata, created_at, sent_at, delivered_at
		FROM notifications
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&notification.ID, &notification.UserID, &notification.Title, &notification.Message,
		&notification.Type, &notification.Priority, &notification.Status, &notification.IsRead,
		&notification.ReadAt, &metadataJSON, &notification.CreatedAt, &notification.SentAt,
		&notification.DeliveredAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("notification not found")
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &notification.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return notification, nil
}

func (r *notificationRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int, filters map[string]interface{}) ([]*models.Notification, int, error) {
	conditions := []string{"user_id = $1"}
	args := []interface{}{userID}
	argCount := 1

	if notificationType, ok := filters["type"].(models.NotificationType); ok {
		argCount++
		conditions = append(conditions, fmt.Sprintf("type = $%d", argCount))
		args = append(args, notificationType)
	}

	if priority, ok := filters["priority"].(models.NotificationPriority); ok {
		argCount++
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argCount))
		args = append(args, priority)
	}

	if isRead, ok := filters["is_read"].(bool); ok {
		argCount++
		conditions = append(conditions, fmt.Sprintf("is_read = $%d", argCount))
		args = append(args, isRead)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			whereClause += " AND " + conditions[i]
		}
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notifications %s", whereClause)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, title, message, type, priority, status, is_read, read_at,
		       metadata, created_at, sent_at, delivered_at
		FROM notifications
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argCount+1, argCount+2)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		notification := &models.Notification{}
		var metadataJSON []byte

		err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.Title, &notification.Message,
			&notification.Type, &notification.Priority, &notification.Status, &notification.IsRead,
			&notification.ReadAt, &metadataJSON, &notification.CreatedAt, &notification.SentAt,
			&notification.DeliveredAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &notification.Metadata); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		notifications = append(notifications, notification)
	}

	return notifications, total, nil
}

func (r *notificationRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notifications
		SET is_read = true, read_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND is_read = false`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found or already read")
	}

	return nil
}

func (r *notificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `
		UPDATE notifications
		SET is_read = true, read_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND is_read = false`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

func (r *notificationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.NotificationStatus) error {
	query := `UPDATE notifications SET status = $2 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}

func (r *notificationRepository) UpdateDeliveryTime(ctx context.Context, id uuid.UUID, deliveredAt time.Time) error {
	query := `UPDATE notifications SET delivered_at = $2, sent_at = COALESCE(sent_at, $2) WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, deliveredAt)
	if err != nil {
		return fmt.Errorf("failed to update delivery time: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}
