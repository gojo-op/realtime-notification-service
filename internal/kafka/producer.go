package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/realtime-notification-system/notification-service/internal/config"
	"github.com/realtime-notification-system/notification-service/internal/models"
)

type NotificationMessage struct {
	NotificationID string                      `json:"notification_id"`
	UserID         string                      `json:"user_id"`
	UserEmail      string                      `json:"user_email"`
	UserPhone      *string                     `json:"user_phone,omitempty"`
	Title          string                      `json:"title"`
	Message        string                      `json:"message"`
	Type           models.NotificationType     `json:"type"`
	Priority       models.NotificationPriority `json:"priority"`
	Metadata       map[string]interface{}      `json:"metadata,omitempty"`
	CreatedAt      time.Time                   `json:"created_at"`
	Attempt        int                         `json:"attempt"`
}

type Producer interface {
	PublishNotification(ctx context.Context, notification *models.Notification, user *models.User) error
	PublishNotificationToDLQ(ctx context.Context, message *NotificationMessage) error
	Close() error
}

type producer struct {
	writer    *kafka.Writer
	dlqWriter *kafka.Writer
	topic     string
	dlqTopic  string
}

func NewProducer(cfg *config.KafkaConfig) (Producer, error) {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.TopicNotifications,
		Balancer:     &kafka.Hash{},
		Compression:  kafka.Snappy,
		RequiredAcks: kafka.RequireAll,
		MaxAttempts:  5,
		BatchSize:    100,
		BatchTimeout: 100 * time.Millisecond,
	}

	dlqWriter := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.TopicNotificationsDLQ,
		Balancer:     &kafka.Hash{},
		Compression:  kafka.Snappy,
		RequiredAcks: kafka.RequireAll,
		MaxAttempts:  5,
	}

	return &producer{
		writer:    writer,
		dlqWriter: dlqWriter,
		topic:     cfg.TopicNotifications,
		dlqTopic:  cfg.TopicNotificationsDLQ,
	}, nil
}

func (p *producer) PublishNotification(ctx context.Context, notification *models.Notification, user *models.User) error {
	message := NotificationMessage{
		NotificationID: notification.ID.String(),
		UserID:         notification.UserID.String(),
		UserEmail:      user.Email,
		Title:          notification.Title,
		Message:        notification.Message,
		Type:           notification.Type,
		Priority:       notification.Priority,
		Metadata:       notification.Metadata,
		CreatedAt:      notification.CreatedAt,
		Attempt:        1,
	}

	if user.Phone != nil {
		message.UserPhone = user.Phone
	}

	value, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal notification message: %w", err)
	}

	kafkaMessage := kafka.Message{
		Key:   []byte(notification.UserID.String()),
		Value: value,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, kafkaMessage); err != nil {
		return fmt.Errorf("failed to publish notification to kafka: %w", err)
	}

	return nil
}

func (p *producer) PublishNotificationToDLQ(ctx context.Context, message *NotificationMessage) error {
	value, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal notification message: %w", err)
	}

	userID, err := uuid.Parse(message.UserID)
	if err != nil {
		return fmt.Errorf("failed to parse user ID: %w", err)
	}

	kafkaMessage := kafka.Message{
		Key:   []byte(userID.String()),
		Value: value,
		Time:  time.Now(),
	}

	if err := p.dlqWriter.WriteMessages(ctx, kafkaMessage); err != nil {
		return fmt.Errorf("failed to publish notification to DLQ: %w", err)
	}

	return nil
}

func (p *producer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("failed to close main writer: %w", err)
	}

	if err := p.dlqWriter.Close(); err != nil {
		return fmt.Errorf("failed to close DLQ writer: %w", err)
	}

	return nil
}
