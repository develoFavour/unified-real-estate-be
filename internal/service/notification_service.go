package service

import (
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/repository"

	"github.com/google/uuid"
)

type NotificationService interface {
	SendNotification(userID, title, message string) error
	GetUserNotifications(userID string) ([]models.Notification, error)
	MarkAsRead(id string) error
}

type notificationService struct {
	repo repository.NotificationRepository
}

func NewNotificationService(repo repository.NotificationRepository) NotificationService {
	return &notificationService{repo}
}

func (s *notificationService) SendNotification(userID, title, message string) error {
	parsedUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	notification := &models.Notification{
		UserID:  parsedUUID,
		Title:   title,
		Message: message,
	}
	return s.repo.CreateNotification(notification)
}

func (s *notificationService) GetUserNotifications(userID string) ([]models.Notification, error) {
	return s.repo.GetUserNotifications(userID)
}

func (s *notificationService) MarkAsRead(id string) error {
	return s.repo.MarkAsRead(id)
}
