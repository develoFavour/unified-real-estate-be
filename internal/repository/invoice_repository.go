package repository

import (
	"real-estate-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvoiceRepository interface {
	Create(invoice *models.Invoice) error
	GetByID(id string) (*models.Invoice, error)
	GetByTenantID(tenantID string) ([]models.Invoice, error)
	GetByPropertyStakeholder(userID string) ([]models.Invoice, error)
	GetPendingByLeaseRequest(leaseRequestID uuid.UUID, invoiceType models.InvoiceType) (*models.Invoice, error)
	Update(invoice *models.Invoice) error
}

type invoiceRepository struct {
	db *gorm.DB
}

func NewInvoiceRepository(db *gorm.DB) InvoiceRepository {
	return &invoiceRepository{db}
}

func (r *invoiceRepository) Create(invoice *models.Invoice) error {
	return r.db.Create(invoice).Error
}

func (r *invoiceRepository) GetByID(id string) (*models.Invoice, error) {
	var invoice models.Invoice
	err := r.db.Preload("Property").Preload("Tenant").Preload("Tenant.Profile").Preload("Lease").Preload("Lease.Documents").Preload("LeaseRequest").
		Where("id = ?", id).First(&invoice).Error
	return &invoice, err
}

func (r *invoiceRepository) GetByTenantID(tenantID string) ([]models.Invoice, error) {
	var invoices []models.Invoice
	err := r.db.Preload("Property").Preload("Lease").Preload("Lease.Documents").Preload("LeaseRequest").
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Find(&invoices).Error
	return invoices, err
}

func (r *invoiceRepository) GetByPropertyStakeholder(userID string) ([]models.Invoice, error) {
	var invoices []models.Invoice
	err := r.db.Preload("Property").Preload("Tenant").Preload("Tenant.Profile").Preload("Lease").Preload("Lease.Documents").Preload("LeaseRequest").
		Joins("JOIN properties ON properties.id = invoices.property_id").
		Where("properties.owner_id = ? OR properties.agent_id = ?", userID, userID).
		Order("invoices.created_at DESC").
		Find(&invoices).Error
	return invoices, err
}

func (r *invoiceRepository) GetPendingByLeaseRequest(leaseRequestID uuid.UUID, invoiceType models.InvoiceType) (*models.Invoice, error) {
	var invoice models.Invoice
	err := r.db.Where("lease_request_id = ? AND type = ? AND status = ?", leaseRequestID, invoiceType, models.InvoicePending).
		First(&invoice).Error
	return &invoice, err
}

func (r *invoiceRepository) Update(invoice *models.Invoice) error {
	return r.db.Save(invoice).Error
}
