package repository

import (
	"real-estate-backend/internal/models"

	"gorm.io/gorm"
)

type MessageRepository interface {
	SaveMessage(message *models.Message) error
	GetChatHistory(user1ID, user2ID string) ([]models.Message, error)
	GetConversations(userID string) ([]models.Message, error)
	MarkAsRead(messageID string) error
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db}
}

func (r *messageRepository) SaveMessage(message *models.Message) error {
	return r.db.Create(message).Error
}

func (r *messageRepository) GetChatHistory(user1ID, user2ID string) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.Where(
		"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		user1ID, user2ID, user2ID, user1ID,
	).Order("created_at ASC").Find(&messages).Error
	return messages, err
}

func (r *messageRepository) GetConversations(userID string) ([]models.Message, error) {
	var messages []models.Message
	// Rank messages by normalized user pair so A->B and B->A are one conversation.
	err := r.db.Preload("Sender.Profile").Preload("Receiver.Profile").
		Where(`
			messages.id IN (
				SELECT id
				FROM (
					SELECT
						id,
						ROW_NUMBER() OVER (
							PARTITION BY LEAST(sender_id::text, receiver_id::text), GREATEST(sender_id::text, receiver_id::text)
							ORDER BY created_at DESC, id DESC
						) AS row_number
					FROM messages
					WHERE deleted_at IS NULL AND (sender_id = ? OR receiver_id = ?)
				) ranked_messages
				WHERE row_number = 1
			)
		`, userID, userID).
		Order("created_at DESC").
		Find(&messages).Error
	return messages, err
}

func (r *messageRepository) MarkAsRead(messageID string) error {
	return r.db.Model(&models.Message{}).Where("id = ?", messageID).Update("is_read", true).Error
}
