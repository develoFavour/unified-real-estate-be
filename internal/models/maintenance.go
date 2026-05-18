package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MaintenanceStatus string
type MaintenancePriority string

const (
	MaintenanceStatusPending      MaintenanceStatus = "PENDING"
	MaintenanceStatusAcknowledged MaintenanceStatus = "ACKNOWLEDGED"
	MaintenanceStatusInProgress   MaintenanceStatus = "IN_PROGRESS"
	MaintenanceStatusResolved     MaintenanceStatus = "RESOLVED"
	MaintenanceStatusClosed       MaintenanceStatus = "CLOSED"
	MaintenanceStatusReopened     MaintenanceStatus = "REOPENED"

	PriorityLow    MaintenancePriority = "LOW"
	PriorityMedium MaintenancePriority = "MEDIUM"
	PriorityHigh   MaintenancePriority = "HIGH"
	PriorityUrgent MaintenancePriority = "URGENT"
)

type MaintenanceRequest struct {
	ID          uuid.UUID           `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PropertyID  uuid.UUID           `gorm:"type:uuid;not null;index;index:idx_maintenance_property_status,priority:1" json:"property_id"`
	TenantID    uuid.UUID           `gorm:"type:uuid;not null;index;index:idx_maintenance_tenant_status,priority:1" json:"tenant_id"`
	Title       string              `gorm:"type:varchar(255);not null" json:"title"`
	Description string              `gorm:"type:text;not null" json:"description"`
	Category    string              `gorm:"type:varchar(50);not null;default:'GENERAL'" json:"category"`
	Priority    MaintenancePriority `gorm:"type:varchar(20);not null;default:'MEDIUM'" json:"priority"`
	ImageURL    string              `gorm:"type:varchar(500)" json:"image_url"`
	Status      MaintenanceStatus   `gorm:"type:varchar(20);default:'PENDING';index:idx_maintenance_property_status,priority:2;index:idx_maintenance_tenant_status,priority:2" json:"status"`
	StatusNote  string              `gorm:"type:text" json:"status_note"`
	CreatedAt   time.Time           `gorm:"index" json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	DeletedAt   gorm.DeletedAt      `gorm:"index" json:"-"`

	Property *Property           `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	Tenant   *User               `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Updates  []MaintenanceUpdate `gorm:"foreignKey:RequestID" json:"updates,omitempty"`
}

type MaintenanceUpdate struct {
	ID        uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	RequestID uuid.UUID         `gorm:"type:uuid;not null;index;index:idx_maintenance_updates_request_created,priority:1" json:"request_id"`
	ActorID   *uuid.UUID        `gorm:"type:uuid;index" json:"actor_id"`
	Status    MaintenanceStatus `gorm:"type:varchar(20);not null" json:"status"`
	Note      string            `gorm:"type:text" json:"note"`
	CreatedAt time.Time         `gorm:"index:idx_maintenance_updates_request_created,priority:2" json:"created_at"`

	Actor *User `gorm:"foreignKey:ActorID" json:"actor,omitempty"`
}
