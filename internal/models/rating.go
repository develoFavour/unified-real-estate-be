package models

import (
	"time"

	"github.com/google/uuid"
)

type Rating struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	AgentID   uuid.UUID `gorm:"type:uuid;not null;index" json:"agent_id"`
	RaterID   uuid.UUID `gorm:"type:uuid;not null" json:"rater_id"`
	Score     int       `gorm:"type:int;not null" json:"score"` // 1-5
	Comment   string    `gorm:"type:text" json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}
