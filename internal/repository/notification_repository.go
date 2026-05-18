package repository

import (
	"real-estate-backend/internal/models"

	"gorm.io/gorm"
)

type NotificationRepository interface {
	CreateNotification(notification *models.Notification) error
	GetUserNotifications(userID string) ([]models.Notification, error)
	MarkAsRead(id string) error
	MarkAllAsRead(userID string) error
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db}
}

func (r *notificationRepository) CreateNotification(notification *models.Notification) error {
	return r.db.Create(notification).Error
}

func (r *notificationRepository) GetUserNotifications(userID string) ([]models.Notification, error) {
	var notifications []models.Notification
	err := r.db.Where("user_id = ?", userID).Order("created_at desc").Find(&notifications).Error
	return notifications, err
}

func (r *notificationRepository) MarkAsRead(id string) error {
	return r.db.Model(&models.Notification{}).Where("id = ?", id).Update("is_read", true).Error
}

func (r *notificationRepository) MarkAllAsRead(userID string) error {
	return r.db.Model(&models.Notification{}).Where("user_id = ?", userID).Update("is_read", true).Error
}
