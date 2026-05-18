package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	SenderID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"sender_id"`
	ReceiverID uuid.UUID      `gorm:"type:uuid;not null;index" json:"receiver_id"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	IsRead     bool           `gorm:"default:false" json:"is_read"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Sender   User `gorm:"foreignKey:SenderID" json:"sender"`
	Receiver User `gorm:"foreignKey:ReceiverID" json:"receiver"`
}
