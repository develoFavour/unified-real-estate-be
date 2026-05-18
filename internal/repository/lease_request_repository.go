package repository

import (
	"real-estate-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LeaseRequestRepository interface {
	Create(req *models.LeaseRequest) error
	GetByID(id string) (*models.LeaseRequest, error)
	GetActiveByTenantAndProperty(tenantID, propertyID uuid.UUID) (*models.LeaseRequest, error)
	GetByTenantID(tenantID string) ([]models.LeaseRequest, error)
	GetIncomingForReviewer(userID string) ([]models.LeaseRequest, error)
	Update(req *models.LeaseRequest) error
}

type leaseRequestRepository struct {
	db *gorm.DB
}

func NewLeaseRequestRepository(db *gorm.DB) LeaseRequestRepository {
	return &leaseRequestRepository{db}
}

func (r *leaseRequestRepository) Create(req *models.LeaseRequest) error {
	return r.db.Create(req).Error
}

func (r *leaseRequestRepository) GetByID(id string) (*models.LeaseRequest, error) {
	var req models.LeaseRequest
	err := r.db.Preload("Property").Preload("Property.Images").
		Preload("Tenant").Preload("Tenant.Profile").
		Preload("Reviewer").Preload("Reviewer.Profile").
		Where("id = ?", id).First(&req).Error
	return &req, err
}

func (r *leaseRequestRepository) GetActiveByTenantAndProperty(tenantID, propertyID uuid.UUID) (*models.LeaseRequest, error) {
	var req models.LeaseRequest
	err := r.db.Where(
		"tenant_id = ? AND property_id = ? AND status IN ?",
		tenantID,
		propertyID,
		[]models.LeaseRequestStatus{models.LeaseRequestPending, models.LeaseRequestApproved},
	).First(&req).Error
	return &req, err
}

func (r *leaseRequestRepository) GetByTenantID(tenantID string) ([]models.LeaseRequest, error) {
	var requests []models.LeaseRequest
	err := r.db.Preload("Property").Preload("Property.Images").
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

func (r *leaseRequestRepository) GetIncomingForReviewer(userID string) ([]models.LeaseRequest, error) {
	var requests []models.LeaseRequest
	err := r.db.Preload("Property").Preload("Property.Images").
		Preload("Tenant").Preload("Tenant.Profile").
		Joins("JOIN properties ON properties.id = lease_requests.property_id").
		Where("(properties.owner_id = ? OR properties.agent_id = ?) AND lease_requests.status = ?", userID, userID, models.LeaseRequestPending).
		Order("lease_requests.created_at DESC").
		Find(&requests).Error
	return requests, err
}

func (r *leaseRequestRepository) Update(req *models.LeaseRequest) error {
	return r.db.Save(req).Error
}
