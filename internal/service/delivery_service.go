package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/realtime-notification-system/notification-service/internal/kafka"
	"github.com/realtime-notification-system/notification-service/internal/models"
	"github.com/realtime-notification-system/notification-service/internal/repository"
)

type DeliveryService interface {
	ProcessNotification(ctx context.Context, message *kafka.NotificationMessage) error
}

type deliveryService struct {
	notificationRepo repository.NotificationRepository
	deliveryLogRepo  repository.DeliveryLogRepository
	userRepo         repository.UserRepository
	producer         kafka.Producer
	maxAttempts      int
}

func NewDeliveryService(notificationRepo repository.NotificationRepository, deliveryLogRepo repository.DeliveryLogRepository, userRepo repository.UserRepository, producer kafka.Producer, maxAttempts int) DeliveryService {
	return &deliveryService{
		notificationRepo: notificationRepo,
		deliveryLogRepo:  deliveryLogRepo,
		userRepo:         userRepo,
		producer:         producer,
		maxAttempts:      maxAttempts,
	}
}

func (s *deliveryService) ProcessNotification(ctx context.Context, message *kafka.NotificationMessage) error {
	startTime := time.Now()

	notificationID, err := uuid.Parse(message.NotificationID)
	if err != nil {
		return fmt.Errorf("invalid notification ID: %w", err)
	}

	userID, err := uuid.Parse(message.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	notification, err := s.notificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return fmt.Errorf("failed to get notification: %w", err)
	}

	processingTime := int(time.Since(startTime).Milliseconds())

	if !s.shouldProcessNotification(notification.Type, user) {
		if err := s.notificationRepo.UpdateStatus(ctx, notificationID, models.NotificationStatusSkipped); err != nil {
			return fmt.Errorf("failed to update notification status to skipped: %w", err)
		}

		deliveryLog := &models.NotificationDeliveryLog{
			NotificationID:   notificationID,
			AttemptNumber:    message.Attempt,
			Status:           models.DeliveryLogStatusSuccess,
			ProcessingTimeMs: &processingTime,
		}

		if err := s.deliveryLogRepo.Create(ctx, deliveryLog); err != nil {
			return fmt.Errorf("failed to create delivery log: %w", err)
		}

		return nil
	}

	delay := s.getProcessingDelay(message.Priority)
	time.Sleep(delay)

	success := s.simulateDelivery()

	if success {
		if err := s.notificationRepo.UpdateStatus(ctx, notificationID, models.NotificationStatusDelivered); err != nil {
			return fmt.Errorf("failed to update notification status to delivered: %w", err)
		}

		if err := s.notificationRepo.UpdateDeliveryTime(ctx, notificationID, time.Now()); err != nil {
			return fmt.Errorf("failed to update delivery time: %w", err)
		}

		deliveryLog := &models.NotificationDeliveryLog{
			NotificationID:   notificationID,
			AttemptNumber:    message.Attempt,
			Status:           models.DeliveryLogStatusSuccess,
			ProcessingTimeMs: &processingTime,
		}

		if err := s.deliveryLogRepo.Create(ctx, deliveryLog); err != nil {
			return fmt.Errorf("failed to create delivery log: %w", err)
		}
	} else {
		if message.Attempt >= s.maxAttempts {
			if err := s.notificationRepo.UpdateStatus(ctx, notificationID, models.NotificationStatusFailed); err != nil {
				return fmt.Errorf("failed to update notification status to failed: %w", err)
			}

			message.Attempt++
			if err := s.producer.PublishNotificationToDLQ(ctx, message); err != nil {
				return fmt.Errorf("failed to publish to DLQ: %w", err)
			}

			errorMsg := "Max retry attempts reached"
			deliveryLog := &models.NotificationDeliveryLog{
				NotificationID:   notificationID,
				AttemptNumber:    message.Attempt,
				Status:           models.DeliveryLogStatusFailed,
				ErrorMessage:     &errorMsg,
				ProcessingTimeMs: &processingTime,
			}

			if err := s.deliveryLogRepo.Create(ctx, deliveryLog); err != nil {
				return fmt.Errorf("failed to create delivery log: %w", err)
			}
		} else {
			message.Attempt++

			delay := s.getRetryDelay(message.Attempt)
			time.Sleep(delay)

			if err := s.producer.PublishNotification(ctx, notification, user); err != nil {
				return fmt.Errorf("failed to republish notification: %w", err)
			}

			deliveryLog := &models.NotificationDeliveryLog{
				NotificationID:   notificationID,
				AttemptNumber:    message.Attempt,
				Status:           models.DeliveryLogStatusRetry,
				ProcessingTimeMs: &processingTime,
			}

			if err := s.deliveryLogRepo.Create(ctx, deliveryLog); err != nil {
				return fmt.Errorf("failed to create delivery log: %w", err)
			}
		}
	}

	return nil
}

func (s *deliveryService) shouldProcessNotification(notificationType models.NotificationType, user *models.User) bool {
	switch notificationType {
	case models.NotificationTypeEmail:
		return user.EmailEnabled
	case models.NotificationTypeSMS:
		return user.SMSEnabled && user.Phone != nil
	case models.NotificationTypePush:
		return user.PushEnabled
	default:
		return false
	}
}

func (s *deliveryService) getProcessingDelay(priority models.NotificationPriority) time.Duration {
	switch priority {
	case models.NotificationPriorityUrgent:
		return 0
	case models.NotificationPriorityHigh:
		return 1 * time.Second
	case models.NotificationPriorityMedium:
		return 2 * time.Second
	case models.NotificationPriorityLow:
		return 5 * time.Second
	default:
		return 2 * time.Second
	}
}

func (s *deliveryService) getRetryDelay(attempt int) time.Duration {
	switch attempt {
	case 2:
		return 30 * time.Second
	case 3:
		return 2 * time.Minute
	default:
		return 30 * time.Second
	}
}

func (s *deliveryService) simulateDelivery() bool {
	return time.Now().UnixNano()%10 < 9 // 90% success rate
}
