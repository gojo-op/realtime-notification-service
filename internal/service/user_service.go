package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/realtime-notification-system/notification-service/internal/models"
	"github.com/realtime-notification-system/notification-service/internal/repository"
)

type UserService interface {
	CreateUser(ctx context.Context, email, phone, fullName string, emailEnabled, smsEnabled, pushEnabled bool) (*models.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateUserPreferences(ctx context.Context, id uuid.UUID, emailEnabled, smsEnabled, pushEnabled bool) error
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) CreateUser(ctx context.Context, email, phone, fullName string, emailEnabled, smsEnabled, pushEnabled bool) (*models.User, error) {
	user := &models.User{
		Email:        email,
		FullName:     fullName,
		EmailEnabled: emailEnabled,
		SMSEnabled:   smsEnabled,
		PushEnabled:  pushEnabled,
	}

	if phone != "" {
		user.Phone = &phone
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *userService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (s *userService) UpdateUserPreferences(ctx context.Context, id uuid.UUID, emailEnabled, smsEnabled, pushEnabled bool) error {
	if err := s.userRepo.UpdatePreferences(ctx, id, emailEnabled, smsEnabled, pushEnabled); err != nil {
		return fmt.Errorf("failed to update user preferences: %w", err)
	}

	return nil
}
