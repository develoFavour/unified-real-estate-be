package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LeaseRequestStatus string

const (
	LeaseRequestPending   LeaseRequestStatus = "PENDING"
	LeaseRequestApproved  LeaseRequestStatus = "APPROVED"
	LeaseRequestRejected  LeaseRequestStatus = "REJECTED"
	LeaseRequestCancelled LeaseRequestStatus = "CANCELLED"
)

type LeaseRequest struct {
	ID         uuid.UUID          `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PropertyID uuid.UUID          `gorm:"type:uuid;not null;index" json:"property_id"`
	TenantID   uuid.UUID          `gorm:"type:uuid;not null;index" json:"tenant_id"`
	ReviewerID *uuid.UUID         `gorm:"type:uuid" json:"reviewer_id"`
	Message    string             `gorm:"type:text" json:"message"`
	Status     LeaseRequestStatus `gorm:"type:varchar(20);default:'PENDING';index" json:"status"`
	ReviewedAt *time.Time         `json:"reviewed_at"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
	DeletedAt  gorm.DeletedAt     `gorm:"index" json:"-"`

	Property *Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	Tenant   *User     `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Reviewer *User     `gorm:"foreignKey:ReviewerID" json:"reviewer,omitempty"`
}
