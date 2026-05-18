package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PropertyOwnershipSource string

const (
	OwnershipSourceListingOwner PropertyOwnershipSource = "LISTING_OWNER"
	OwnershipSourceSale         PropertyOwnershipSource = "SALE"
	OwnershipSourceAdmin        PropertyOwnershipSource = "ADMIN"
)

type PropertyOwnership struct {
	ID                uuid.UUID               `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PropertyID        uuid.UUID               `gorm:"type:uuid;not null;index;index:idx_property_current_ownership,priority:1" json:"property_id"`
	UserID            uuid.UUID               `gorm:"type:uuid;not null;index" json:"user_id"`
	SaleReservationID *uuid.UUID              `gorm:"type:uuid;index" json:"sale_reservation_id"`
	Source            PropertyOwnershipSource `gorm:"type:varchar(40);not null;index" json:"source"`
	IsCurrent         bool                    `gorm:"default:true;index:idx_property_current_ownership,priority:2" json:"is_current"`
	StartedAt         time.Time               `gorm:"not null" json:"started_at"`
	EndedAt           *time.Time              `json:"ended_at"`
	Notes             string                  `gorm:"type:text" json:"notes"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
	DeletedAt         gorm.DeletedAt          `gorm:"index" json:"-"`

	Property        *Property        `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	User            *User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	SaleReservation *SaleReservation `gorm:"foreignKey:SaleReservationID" json:"sale_reservation,omitempty"`
}
