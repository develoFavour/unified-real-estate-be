package models

import (
	"time"

	"github.com/google/uuid"
)

type SavedProperty struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	PropertyID uuid.UUID `gorm:"type:uuid;index;not null" json:"property_id"`
	CreatedAt  time.Time `json:"created_at"`

	User     *User     `gorm:"foreignKey:UserID" json:"-"`
	Property *Property `gorm:"foreignKey:PropertyID" json:"property"`
}
