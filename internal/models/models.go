package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	Phone        *string   `db:"phone" json:"phone,omitempty"`
	FullName     string    `db:"full_name" json:"full_name"`
	EmailEnabled bool      `db:"email_enabled" json:"email_enabled"`
	SMSEnabled   bool      `db:"sms_enabled" json:"sms_enabled"`
	PushEnabled  bool      `db:"push_enabled" json:"push_enabled"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type NotificationType string

const (
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeSMS   NotificationType = "sms"
	NotificationTypePush  NotificationType = "push"
)

type NotificationPriority string

const (
	NotificationPriorityLow    NotificationPriority = "low"
	NotificationPriorityMedium NotificationPriority = "medium"
	NotificationPriorityHigh   NotificationPriority = "high"
	NotificationPriorityUrgent NotificationPriority = "urgent"
)

type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusDelivered NotificationStatus = "delivered"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusSkipped   NotificationStatus = "skipped"
)

type Notification struct {
	ID          uuid.UUID              `db:"id" json:"id"`
	UserID      uuid.UUID              `db:"user_id" json:"user_id"`
	Title       string                 `db:"title" json:"title"`
	Message     string                 `db:"message" json:"message"`
	Type        NotificationType       `db:"type" json:"type"`
	Priority    NotificationPriority   `db:"priority" json:"priority"`
	Status      NotificationStatus     `db:"status" json:"status"`
	IsRead      bool                   `db:"is_read" json:"is_read"`
	ReadAt      *time.Time             `db:"read_at" json:"read_at,omitempty"`
	Metadata    map[string]interface{} `db:"metadata" json:"metadata,omitempty"`
	CreatedAt   time.Time              `db:"created_at" json:"created_at"`
	SentAt      *time.Time             `db:"sent_at" json:"sent_at,omitempty"`
	DeliveredAt *time.Time             `db:"delivered_at" json:"delivered_at,omitempty"`
}

type DeliveryLogStatus string

const (
	DeliveryLogStatusSuccess DeliveryLogStatus = "success"
	DeliveryLogStatusFailed  DeliveryLogStatus = "failed"
	DeliveryLogStatusRetry   DeliveryLogStatus = "retry"
)

type NotificationDeliveryLog struct {
	ID               uuid.UUID         `db:"id" json:"id"`
	NotificationID   uuid.UUID         `db:"notification_id" json:"notification_id"`
	AttemptNumber    int               `db:"attempt_number" json:"attempt_number"`
	Status           DeliveryLogStatus `db:"status" json:"status"`
	ErrorMessage     *string           `db:"error_message" json:"error_message,omitempty"`
	ProcessingTimeMs *int              `db:"processing_time_ms" json:"processing_time_ms,omitempty"`
	CreatedAt        time.Time         `db:"created_at" json:"created_at"`
}
