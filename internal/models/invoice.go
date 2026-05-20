package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvoiceType string
type InvoiceStatus string

const (
	InvoiceReservationDeposit InvoiceType = "RESERVATION_DEPOSIT"
	InvoiceRent               InvoiceType = "RENT"
	InvoiceCautionDeposit     InvoiceType = "CAUTION_DEPOSIT"
	InvoiceAgencyFee          InvoiceType = "AGENCY_FEE"
	InvoiceLegalFee           InvoiceType = "LEGAL_FEE"
	InvoiceServiceCharge      InvoiceType = "SERVICE_CHARGE"

	InvoicePending   InvoiceStatus = "PENDING"
	InvoicePaid      InvoiceStatus = "PAID"
	InvoiceCancelled InvoiceStatus = "CANCELLED"
	InvoiceOverdue   InvoiceStatus = "OVERDUE"
)

type Invoice struct {
	ID                 uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantID           uuid.UUID      `gorm:"type:uuid;not null;index" json:"tenant_id"`
	PropertyID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"property_id"`
	LeaseID            *uuid.UUID     `gorm:"type:uuid;index" json:"lease_id"`
	LeaseRequestID     *uuid.UUID     `gorm:"type:uuid;index" json:"lease_request_id"`
	Type               InvoiceType    `gorm:"type:varchar(40);not null;index" json:"type"`
	Amount             float64        `gorm:"type:numeric(12,2);not null" json:"amount"`
	Status             InvoiceStatus  `gorm:"type:varchar(20);default:'PENDING';index" json:"status"`
	DueDate            *time.Time     `json:"due_date"`
	BillingPeriodStart *time.Time     `json:"billing_period_start"`
	BillingPeriodEnd   *time.Time     `json:"billing_period_end"`
	PaidAt             *time.Time     `json:"paid_at"`
	PaymentReference   string         `gorm:"type:varchar(100);index" json:"payment_reference"`
	Description        string         `gorm:"type:text" json:"description"`
	CreatedAt          time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`

	Tenant       *User           `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Property     *Property       `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	Lease        *LeaseAgreement `gorm:"foreignKey:LeaseID" json:"lease,omitempty"`
	LeaseRequest *LeaseRequest   `gorm:"foreignKey:LeaseRequestID" json:"lease_request,omitempty"`
}
