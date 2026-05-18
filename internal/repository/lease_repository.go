package repository

import (
	"real-estate-backend/internal/models"

	"gorm.io/gorm"
)

type LeaseRepository interface {
	CreateLease(lease *models.LeaseAgreement) error
	GetByTenantID(tenantID string) (*models.LeaseAgreement, error)
	GetByPropertyID(propertyID string) (*models.LeaseAgreement, error)
	GetLeaseByID(id string) (*models.LeaseAgreement, error)
	UpdateLease(lease *models.LeaseAgreement) error
}

type leaseRepository struct {
	db *gorm.DB
}

func NewLeaseRepository(db *gorm.DB) LeaseRepository {
	return &leaseRepository{db}
}

func (r *leaseRepository) CreateLease(lease *models.LeaseAgreement) error {
	return r.db.Create(lease).Error
}

func (r *leaseRepository) GetByTenantID(tenantID string) (*models.LeaseAgreement, error) {
	var lease models.LeaseAgreement
	err := r.db.Where("tenant_id = ? AND status = ?", tenantID, models.LeaseStatusActive).First(&lease).Error
	if err != nil {
		return nil, err
	}
	return &lease, nil
}

func (r *leaseRepository) GetByPropertyID(propertyID string) (*models.LeaseAgreement, error) {
	var lease models.LeaseAgreement
	err := r.db.Where("property_id = ? AND status = ?", propertyID, models.LeaseStatusActive).First(&lease).Error
	if err != nil {
		return nil, err
	}
	return &lease, nil
}

func (r *leaseRepository) GetLeaseByID(id string) (*models.LeaseAgreement, error) {
	var lease models.LeaseAgreement
	err := r.db.Where("id = ?", id).First(&lease).Error
	return &lease, err
}

func (r *leaseRepository) UpdateLease(lease *models.LeaseAgreement) error {
	return r.db.Save(lease).Error
}
