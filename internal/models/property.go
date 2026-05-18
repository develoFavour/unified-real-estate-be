package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PropertyStatus string
type ListingType string

const (
	StatusAvailable   PropertyStatus = "AVAILABLE"
	StatusRented      PropertyStatus = "RENTED"
	StatusReserved    PropertyStatus = "RESERVED"
	StatusUnderReview PropertyStatus = "UNDER_REVIEW"
	StatusSold        PropertyStatus = "SOLD"
	StatusCancelled   PropertyStatus = "CANCELLED"
	StatusMaintenance PropertyStatus = "MAINTENANCE"

	TypeRent ListingType = "RENT"
	TypeSale ListingType = "SALE"
)

type Property struct {
	ID                    uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OwnerID               *uuid.UUID      `gorm:"type:uuid" json:"owner_id"`
	AgentID               *uuid.UUID      `gorm:"type:uuid;index" json:"agent_id"`
	AgentAssignmentStatus string          `gorm:"type:varchar(20);default:'NONE'" json:"agent_assignment_status"` // NONE, PENDING, ACCEPTED, REJECTED
	PropertyType          string          `gorm:"type:varchar(50);index" json:"property_type"`
	ListingType           ListingType     `gorm:"type:varchar(20);default:'RENT';index" json:"listing_type"`
	Title                 string          `gorm:"type:varchar(255);not null" json:"title"`
	Description           string          `gorm:"type:text;not null" json:"description"`
	Price                 float64         `gorm:"type:numeric(12,2);not null;index" json:"price"` // Annual Rent or Sale Price
	TotalSalePrice        float64         `gorm:"type:numeric(12,2)" json:"total_sale_price"`
	MinimumHoldingFee     float64         `gorm:"type:numeric(12,2)" json:"minimum_holding_fee"`
	IsOffPlan             bool            `gorm:"default:false" json:"is_off_plan"`
	AgencyFee             float64         `gorm:"type:numeric(12,2)" json:"agency_fee"`
	LegalFee              float64         `gorm:"type:numeric(12,2)" json:"legal_fee"`
	CautionDeposit        float64         `gorm:"type:numeric(12,2)" json:"caution_deposit"`
	TotalPackage          float64         `gorm:"type:numeric(12,2)" json:"total_package"`
	MandateType           string          `gorm:"type:varchar(50)" json:"mandate_type"` // EXCLUSIVE or SHARED
	MandateDocumentURL    string          `gorm:"type:varchar(255)" json:"mandate_document_url"`
	Status                PropertyStatus  `gorm:"type:varchar(20);default:'AVAILABLE';index" json:"status"`
	IsVerified            bool            `gorm:"default:false" json:"is_verified"`
	Address               string          `gorm:"type:text;not null" json:"address"`
	City                  string          `gorm:"type:varchar(100);not null;index" json:"city"`
	State                 string          `gorm:"type:varchar(100);not null;index" json:"state"`
	Latitude              float64         `gorm:"type:numeric(10,8)" json:"latitude"`
	Longitude             float64         `gorm:"type:numeric(11,8)" json:"longitude"`
	Bedrooms              *int            `json:"bedrooms"`
	Bathrooms             *int            `json:"bathrooms"`
	SquareFeet            *float64        `json:"square_feet"`
	YearBuilt             *int            `json:"year_built"`
	Amenities             string          `gorm:"type:text" json:"amenities"` // Comma separated list
	Images                []PropertyImage `gorm:"foreignKey:PropertyID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"images"`
	Owner                 *User           `gorm:"foreignKey:OwnerID" json:"owner"`
	Agent                 *User           `gorm:"foreignKey:AgentID" json:"agent"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
	DeletedAt             gorm.DeletedAt  `gorm:"index" json:"-"`
}

type PropertyImage struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PropertyID uuid.UUID `gorm:"type:uuid;not null;index" json:"property_id"`
	ImageURL   string    `gorm:"type:varchar(255);not null" json:"image_url"`
	IsPrimary  bool      `gorm:"default:false" json:"is_primary"`
	CreatedAt  time.Time `json:"created_at"`
}
