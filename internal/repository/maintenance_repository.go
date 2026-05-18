package repository

import (
	"context"
	"real-estate-backend/internal/models"
	"time"

	"gorm.io/gorm"
)

const maintenanceQueryTimeout = 8 * time.Second

type MaintenanceRepository interface {
	CreateRequest(request *models.MaintenanceRequest) error
	GetRequestByID(id string) (*models.MaintenanceRequest, error)
	GetByOwnerID(ownerID string) ([]models.MaintenanceRequest, error)
	GetByAgentID(agentID string) ([]models.MaintenanceRequest, error)
	GetByPropertyID(propertyID string) ([]models.MaintenanceRequest, error)
	GetByTenantID(tenantID string) ([]models.MaintenanceRequest, error)
	UpdateStatus(id string, status models.MaintenanceStatus, note string) error
	AddUpdate(update *models.MaintenanceUpdate) error
}

type maintenanceRepository struct {
	db *gorm.DB
}

func NewMaintenanceRepository(db *gorm.DB) MaintenanceRepository {
	return &maintenanceRepository{db}
}

func (r *maintenanceRepository) CreateRequest(request *models.MaintenanceRequest) error {
	db, cancel := r.withTimeout()
	defer cancel()
	return db.Create(request).Error
}

func (r *maintenanceRepository) GetRequestByID(id string) (*models.MaintenanceRequest, error) {
	var request models.MaintenanceRequest
	db, cancel := r.withTimeout()
	defer cancel()
	err := db.First(&request, "id = ?", id).Error
	return &request, err
}

func (r *maintenanceRepository) GetByOwnerID(ownerID string) ([]models.MaintenanceRequest, error) {
	var requests []models.MaintenanceRequest
	db, cancel := r.withTimeout()
	defer cancel()
	// Join MaintenanceRequests with Properties to filter by OwnerID
	err := r.listPreload(db).Table("maintenance_requests").
		Select("maintenance_requests.*").
		Joins("JOIN properties ON properties.id = maintenance_requests.property_id").
		Where("properties.owner_id = ?", ownerID).
		Order("maintenance_requests.created_at DESC").
		Find(&requests).Error
	return requests, err
}

func (r *maintenanceRepository) GetByAgentID(agentID string) ([]models.MaintenanceRequest, error) {
	var requests []models.MaintenanceRequest
	db, cancel := r.withTimeout()
	defer cancel()
	// Join MaintenanceRequests with Properties to filter by AgentID
	err := r.listPreload(db).Table("maintenance_requests").
		Select("maintenance_requests.*").
		Joins("JOIN properties ON properties.id = maintenance_requests.property_id").
		Where("properties.agent_id = ?", agentID).
		Order("maintenance_requests.created_at DESC").
		Find(&requests).Error
	return requests, err
}

func (r *maintenanceRepository) GetByPropertyID(propertyID string) ([]models.MaintenanceRequest, error) {
	var requests []models.MaintenanceRequest
	db, cancel := r.withTimeout()
	defer cancel()
	err := r.listPreload(db).Where("property_id = ?", propertyID).Order("created_at desc").Find(&requests).Error
	return requests, err
}

func (r *maintenanceRepository) GetByTenantID(tenantID string) ([]models.MaintenanceRequest, error) {
	var requests []models.MaintenanceRequest
	db, cancel := r.withTimeout()
	defer cancel()
	err := r.listPreload(db).Where("tenant_id = ?", tenantID).Order("created_at desc").Find(&requests).Error
	return requests, err
}

func (r *maintenanceRepository) UpdateStatus(id string, status models.MaintenanceStatus, note string) error {
	db, cancel := r.withTimeout()
	defer cancel()
	return db.Model(&models.MaintenanceRequest{}).Where("id = ?", id).Updates(map[string]any{
		"status":      status,
		"status_note": note,
	}).Error
}

func (r *maintenanceRepository) AddUpdate(update *models.MaintenanceUpdate) error {
	db, cancel := r.withTimeout()
	defer cancel()
	return db.Create(update).Error
}

func (r *maintenanceRepository) listPreload(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Updates", func(db *gorm.DB) *gorm.DB {
			return db.Order("maintenance_updates.created_at ASC")
		})
}

func (r *maintenanceRepository) withTimeout() (*gorm.DB, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), maintenanceQueryTimeout)
	return r.db.WithContext(ctx), cancel
}
