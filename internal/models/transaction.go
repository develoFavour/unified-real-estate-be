package models

import (
	"time"

	"github.com/google/uuid"
)

type PropertyPaymentType string
type PropertyPaymentStatus string

const (
	PayHoldingFee   PropertyPaymentType = "HOLDING_FEE"
	PayFullPayment  PropertyPaymentType = "FULL_PAYMENT"
	PayRent         PropertyPaymentType = "RENT"
	PayMilestone    PropertyPaymentType = "MILESTONE"

	PayStatusPending   PropertyPaymentStatus = "PENDING"
	PayStatusEscrow    PropertyPaymentStatus = "ESCROW"
	PayStatusCompleted PropertyPaymentStatus = "COMPLETED"
	PayStatusReversed  PropertyPaymentStatus = "REVERSED"
)

type PropertyTransaction struct {
	ID          uuid.UUID             `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BuyerID     uuid.UUID             `gorm:"type:uuid;index;not null" json:"buyer_id"`
	PropertyID  uuid.UUID             `gorm:"type:uuid;index;not null" json:"property_id"`
	Amount      float64               `gorm:"type:numeric(12,2);not null" json:"amount"`
	Type        PropertyPaymentType   `gorm:"type:varchar(20);not null" json:"type"`
	Status      PropertyPaymentStatus `gorm:"type:varchar(20);default:'PENDING'" json:"status"`
	Reference   string        `gorm:"type:varchar(100);uniqueIndex" json:"reference"`
	EscrowUntil *time.Time    `json:"escrow_until"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`

	Buyer    *User     `gorm:"foreignKey:BuyerID" json:"buyer,omitempty"`
	Property *Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
}

type PaymentMilestone struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PropertyID  uuid.UUID `gorm:"type:uuid;index;not null" json:"property_id"`
	Title       string    `gorm:"type:varchar(255);not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	Percentage  float64   `gorm:"type:numeric(5,2);not null" json:"percentage"`
	Amount      float64   `gorm:"type:numeric(12,2);not null" json:"amount"`
	Order       int       `json:"order"`
	IsPaid      bool      `gorm:"default:false" json:"is_paid"`
	CreatedAt   time.Time `json:"created_at"`
}
