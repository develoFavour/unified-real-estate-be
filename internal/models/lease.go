package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LeaseStatus string

const (
	LeaseStatusPending    LeaseStatus = "PENDING"
	LeaseStatusActive     LeaseStatus = "ACTIVE"
	LeaseStatusExpired    LeaseStatus = "EXPIRED"
	LeaseStatusTerminated LeaseStatus = "TERMINATED"
)

type LeaseAgreement struct {
	ID             uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PropertyID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"property_id"`
	TenantID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"tenant_id"`
	StartDate      time.Time      `gorm:"not null" json:"start_date"`
	EndDate        time.Time      `gorm:"not null" json:"end_date"`
	RentAmount     float64        `gorm:"type:numeric(12,2);not null" json:"rent_amount"`
	DocumentURL    string         `gorm:"type:varchar(255)" json:"document_url"`
	TenantAccepted bool           `gorm:"default:false" json:"tenant_accepted"`
	Status         LeaseStatus    `gorm:"type:varchar(20);default:'PENDING'" json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	Property  *Property             `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	Tenant    *User                 `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Documents []TransactionDocument `gorm:"foreignKey:LeaseID" json:"documents,omitempty"`
}
