package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "PENDING"
	PaymentStatusSuccess PaymentStatus = "SUCCESS"
	PaymentStatusFailed  PaymentStatus = "FAILED"
)

type Payment struct {
	ID               uuid.UUID     `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	LeaseID          uuid.UUID     `gorm:"type:uuid;not null;index" json:"lease_id"`
	TenantID         uuid.UUID     `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Amount           float64       `gorm:"type:numeric(12,2);not null" json:"amount"`
	PaymentReference string        `gorm:"type:varchar(100);uniqueIndex" json:"payment_reference"`
	PaymentDate      *time.Time    `json:"payment_date"`
	Status           PaymentStatus `gorm:"type:varchar(20);default:'PENDING'" json:"status"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}
