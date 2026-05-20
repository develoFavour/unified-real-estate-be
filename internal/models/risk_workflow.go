package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SaleReservationStatus string
type TransactionDocumentType string
type DisputeStatus string
type CaseType string
type CasePriority string

const (
	SaleReservationReserved    SaleReservationStatus = "RESERVED"
	SaleReservationUnderReview SaleReservationStatus = "UNDER_REVIEW"
	SaleReservationAccepted    SaleReservationStatus = "ACCEPTED"
	SaleReservationDisputed    SaleReservationStatus = "DISPUTED"
	SaleReservationCancelled   SaleReservationStatus = "CANCELLED"
	SaleReservationExpired     SaleReservationStatus = "EXPIRED"
	SaleReservationRefunded    SaleReservationStatus = "REFUNDED"
	SaleReservationSold        SaleReservationStatus = "SOLD"

	DocumentLeaseAgreement TransactionDocumentType = "LEASE_AGREEMENT"
	DocumentInspection     TransactionDocumentType = "INSPECTION"
	DocumentHandoverNote   TransactionDocumentType = "HANDOVER_NOTE"
	DocumentIDVerification TransactionDocumentType = "ID_VERIFICATION"
	DocumentTitle          TransactionDocumentType = "TITLE_DOCUMENT"
	DocumentOwnershipProof TransactionDocumentType = "OWNERSHIP_PROOF"
	DocumentSaleAgreement  TransactionDocumentType = "SALE_AGREEMENT"
	DocumentLegal          TransactionDocumentType = "LEGAL_DOCUMENT"
	DocumentOther          TransactionDocumentType = "OTHER"

	DisputeOpen        DisputeStatus = "OPEN"
	DisputeResponded   DisputeStatus = "RESPONDED"
	DisputeUnderReview DisputeStatus = "UNDER_REVIEW"
	DisputeResolved    DisputeStatus = "RESOLVED"
	DisputeRejected    DisputeStatus = "REJECTED"

	CaseTypeSupport CaseType = "SUPPORT"
	CaseTypeDispute CaseType = "DISPUTE"

	CasePriorityLow    CasePriority = "LOW"
	CasePriorityMedium CasePriority = "MEDIUM"
	CasePriorityHigh   CasePriority = "HIGH"
	CasePriorityUrgent CasePriority = "URGENT"
)

type SaleReservation struct {
	ID                       uuid.UUID             `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BuyerID                  uuid.UUID             `gorm:"type:uuid;not null;index" json:"buyer_id"`
	PropertyID               uuid.UUID             `gorm:"type:uuid;not null;index" json:"property_id"`
	WalletTransactionID      *uuid.UUID            `gorm:"type:uuid;index" json:"wallet_transaction_id"`
	PaymentReference         string                `gorm:"type:varchar(120);uniqueIndex;not null" json:"payment_reference"`
	Amount                   float64               `gorm:"type:numeric(12,2);not null" json:"amount"`
	Status                   SaleReservationStatus `gorm:"type:varchar(30);default:'RESERVED';index;index:idx_sale_reservations_status_created,priority:1" json:"status"`
	ExpiresAt                *time.Time            `json:"expires_at"`
	BuyerAcceptedAt          *time.Time            `json:"buyer_accepted_at"`
	FinalSettlementAmount    float64               `gorm:"type:numeric(12,2);default:0" json:"final_settlement_amount"`
	FinalSettlementReference string                `gorm:"type:varchar(120)" json:"final_settlement_reference"`
	FinalSettlementDate      *time.Time            `json:"final_settlement_date"`
	FinalSettlementNotes     string                `gorm:"type:text" json:"final_settlement_notes"`
	CreatedAt                time.Time             `gorm:"index;index:idx_sale_reservations_status_created,priority:2" json:"created_at"`
	UpdatedAt                time.Time             `json:"updated_at"`
	DeletedAt                gorm.DeletedAt        `gorm:"index" json:"-"`

	Buyer             *User                 `gorm:"foreignKey:BuyerID" json:"buyer,omitempty"`
	Property          *Property             `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	WalletTransaction *WalletTransaction    `gorm:"foreignKey:WalletTransactionID" json:"wallet_transaction,omitempty"`
	Documents         []TransactionDocument `gorm:"foreignKey:SaleReservationID" json:"documents,omitempty"`
}

type TransactionDocument struct {
	ID                uuid.UUID               `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UploadedByID      uuid.UUID               `gorm:"type:uuid;not null;index" json:"uploaded_by_id"`
	PropertyID        *uuid.UUID              `gorm:"type:uuid;index" json:"property_id"`
	LeaseID           *uuid.UUID              `gorm:"type:uuid;index" json:"lease_id"`
	InvoiceID         *uuid.UUID              `gorm:"type:uuid;index" json:"invoice_id"`
	SaleReservationID *uuid.UUID              `gorm:"type:uuid;index" json:"sale_reservation_id"`
	Type              TransactionDocumentType `gorm:"type:varchar(40);not null;index" json:"type"`
	Title             string                  `gorm:"type:varchar(255);not null" json:"title"`
	DocumentURL       string                  `gorm:"type:varchar(500);not null" json:"document_url"`
	Notes             string                  `gorm:"type:text" json:"notes"`
	CreatedAt         time.Time               `gorm:"index" json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
	DeletedAt         gorm.DeletedAt          `gorm:"index" json:"-"`

	UploadedBy      *User            `gorm:"foreignKey:UploadedByID" json:"uploaded_by,omitempty"`
	Property        *Property        `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	Lease           *LeaseAgreement  `gorm:"foreignKey:LeaseID" json:"lease,omitempty"`
	Invoice         *Invoice         `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
	SaleReservation *SaleReservation `gorm:"foreignKey:SaleReservationID" json:"sale_reservation,omitempty"`
}

type Dispute struct {
	ID                  uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OpenedByID          uuid.UUID      `gorm:"type:uuid;not null;index" json:"opened_by_id"`
	RespondentID        *uuid.UUID     `gorm:"type:uuid;index" json:"respondent_id"`
	ReportedUserID      *uuid.UUID     `gorm:"type:uuid;index" json:"reported_user_id"`
	PropertyID          *uuid.UUID     `gorm:"type:uuid;index" json:"property_id"`
	InvoiceID           *uuid.UUID     `gorm:"type:uuid;index" json:"invoice_id"`
	SaleReservationID   *uuid.UUID     `gorm:"type:uuid;index" json:"sale_reservation_id"`
	WalletTransactionID *uuid.UUID     `gorm:"type:uuid;index" json:"wallet_transaction_id"`
	CaseType            CaseType       `gorm:"type:varchar(20);default:'DISPUTE';index:idx_disputes_case_status_created,priority:1" json:"case_type"`
	Category            string         `gorm:"type:varchar(80);default:'GENERAL';index" json:"category"`
	Title               string         `gorm:"type:varchar(160)" json:"title"`
	Priority            CasePriority   `gorm:"type:varchar(20);default:'MEDIUM';index" json:"priority"`
	Reason              string         `gorm:"type:varchar(120);not null" json:"reason"`
	Description         string         `gorm:"type:text;not null" json:"description"`
	EvidenceURL         string         `gorm:"type:varchar(500)" json:"evidence_url"`
	Status              DisputeStatus  `gorm:"type:varchar(30);default:'OPEN';index;index:idx_disputes_status_created,priority:1;index:idx_disputes_case_status_created,priority:2" json:"status"`
	Response            string         `gorm:"type:text" json:"response"`
	Resolution          string         `gorm:"type:text" json:"resolution"`
	CreatedAt           time.Time      `gorm:"index;index:idx_disputes_status_created,priority:2;index:idx_disputes_case_status_created,priority:3" json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`

	OpenedBy          *User              `gorm:"foreignKey:OpenedByID" json:"opened_by,omitempty"`
	Respondent        *User              `gorm:"foreignKey:RespondentID" json:"respondent,omitempty"`
	ReportedUser      *User              `gorm:"foreignKey:ReportedUserID" json:"reported_user,omitempty"`
	Property          *Property          `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	Invoice           *Invoice           `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
	SaleReservation   *SaleReservation   `gorm:"foreignKey:SaleReservationID" json:"sale_reservation,omitempty"`
	WalletTransaction *WalletTransaction `gorm:"foreignKey:WalletTransactionID" json:"wallet_transaction,omitempty"`
}

type AuditEvent struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ActorID    *uuid.UUID     `gorm:"type:uuid;index" json:"actor_id"`
	EntityType string         `gorm:"type:varchar(80);not null;index" json:"entity_type"`
	EntityID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"entity_id"`
	Action     string         `gorm:"type:varchar(120);not null;index" json:"action"`
	Details    string         `gorm:"type:text" json:"details"`
	CreatedAt  time.Time      `gorm:"index" json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	Actor *User `gorm:"foreignKey:ActorID" json:"actor,omitempty"`
}
