package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/realtime-notification-system/notification-service/internal/kafka"
	"github.com/realtime-notification-system/notification-service/internal/models"
	"github.com/realtime-notification-system/notification-service/internal/repository"
)

type NotificationService interface {
	CreateNotification(ctx context.Context, userID uuid.UUID, title, message string, notificationType models.NotificationType, priority models.NotificationPriority, metadata map[string]interface{}) (*models.Notification, error)
	CreateBatchNotifications(ctx context.Context, userIDs []uuid.UUID, title, message string, notificationType models.NotificationType, priority models.NotificationPriority, metadata map[string]interface{}) (int, error)
	GetNotification(ctx context.Context, id uuid.UUID) (*models.Notification, error)
	GetUserNotifications(ctx context.Context, userID uuid.UUID, page, limit int, filters map[string]interface{}) ([]*models.Notification, int, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error)
}

type notificationService struct {
	notificationRepo repository.NotificationRepository
	userRepo         repository.UserRepository
	producer         kafka.Producer
}

func NewNotificationService(notificationRepo repository.NotificationRepository, userRepo repository.UserRepository, producer kafka.Producer) NotificationService {
	return &notificationService{
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		producer:         producer,
	}
}

func (s *notificationService) CreateNotification(ctx context.Context, userID uuid.UUID, title, message string, notificationType models.NotificationType, priority models.NotificationPriority, metadata map[string]interface{}) (*models.Notification, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	notification := &models.Notification{
		UserID:   userID,
		Title:    title,
		Message:  message,
		Type:     notificationType,
		Priority: priority,
		Metadata: metadata,
	}

	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	if err := s.producer.PublishNotification(ctx, notification, user); err != nil {
		return nil, fmt.Errorf("failed to publish notification to kafka: %w", err)
	}

	return notification, nil
}

func (s *notificationService) CreateBatchNotifications(ctx context.Context, userIDs []uuid.UUID, title, message string, notificationType models.NotificationType, priority models.NotificationPriority, metadata map[string]interface{}) (int, error) {
	var notifications []*models.Notification

	for _, userID := range userIDs {
		notification := &models.Notification{
			UserID:   userID,
			Title:    title,
			Message:  message,
			Type:     notificationType,
			Priority: priority,
			Metadata: metadata,
		}
		notifications = append(notifications, notification)
	}

	if err := s.notificationRepo.CreateBatch(ctx, notifications); err != nil {
		return 0, fmt.Errorf("failed to create batch notifications: %w", err)
	}

	for _, notification := range notifications {
		user, err := s.userRepo.GetByID(ctx, notification.UserID)
		if err != nil {
			return 0, fmt.Errorf("failed to get user %s: %w", notification.UserID, err)
		}

		if err := s.producer.PublishNotification(ctx, notification, user); err != nil {
			return 0, fmt.Errorf("failed to publish notification to kafka: %w", err)
		}
	}

	return len(notifications), nil
}

func (s *notificationService) GetNotification(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	notification, err := s.notificationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return notification, nil
}

func (s *notificationService) GetUserNotifications(ctx context.Context, userID uuid.UUID, page, limit int, filters map[string]interface{}) ([]*models.Notification, int, error) {
	offset := (page - 1) * limit
	notifications, total, err := s.notificationRepo.GetByUserID(ctx, userID, limit, offset, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user notifications: %w", err)
	}

	return notifications, total, nil
}

func (s *notificationService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := s.notificationRepo.GetUnreadCount(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

func (s *notificationService) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	if err := s.notificationRepo.MarkAsRead(ctx, id); err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}

func (s *notificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.notificationRepo.MarkAllAsRead(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	return count, nil
}
