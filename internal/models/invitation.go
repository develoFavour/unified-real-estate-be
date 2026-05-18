package models

import (
	"time"

	"github.com/google/uuid"
)

type InvitationStatus string

const (
	InvitationPending   InvitationStatus = "PENDING"
	InvitationAccepted  InvitationStatus = "ACCEPTED"
	InvitationExpired   InvitationStatus = "EXPIRED"
)

type Invitation struct {
	ID         uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InviterID  uuid.UUID        `gorm:"type:uuid;not null" json:"inviter_id"`
	Email      string           `gorm:"type:varchar(255);not null" json:"email"`
	Role       Role             `gorm:"type:varchar(20);not null" json:"role"`
	PropertyID *uuid.UUID       `gorm:"type:uuid" json:"property_id"` // Optional: Link to a specific property
	Token      string           `gorm:"type:varchar(255);unique;not null" json:"token"`
	Status     InvitationStatus `gorm:"type:varchar(20);default:'PENDING'" json:"status"`
	ExpiresAt  time.Time        `json:"expires_at"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}
