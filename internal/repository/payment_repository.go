package repository

import (
	"real-estate-backend/internal/models"
	"gorm.io/gorm"
)

type PaymentRepository interface {
	CreatePayment(payment *models.Payment) error
	GetByOwnerID(ownerID string) ([]models.Payment, error)
	GetByTenantID(tenantID string) ([]models.Payment, error)
	GetMonthlyIncome(ownerID string) (any, error)
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db}
}

func (r *paymentRepository) CreatePayment(payment *models.Payment) error {
	return r.db.Create(payment).Error
}

func (r *paymentRepository) GetByOwnerID(ownerID string) ([]models.Payment, error) {
	var payments []models.Payment
	err := r.db.Table("payments").
		Select("payments.*").
		Joins("JOIN lease_agreements ON lease_agreements.id = payments.lease_id").
		Joins("JOIN properties ON properties.id = lease_agreements.property_id").
		Where("properties.owner_id = ? AND payments.status = ?", ownerID, models.PaymentStatusSuccess).
		Order("payments.payment_date DESC").
		Find(&payments).Error
	return payments, err
}

func (r *paymentRepository) GetMonthlyIncome(ownerID string) (any, error) {
	type Result struct {
		Month string  `json:"month"`
		Total float64 `json:"total"`
	}
	var results []Result

	// Get last 6 months of successful payments
	err := r.db.Table("payments").
		Select("TO_CHAR(payments.payment_date, 'Mon') as month, SUM(payments.amount) as total").
		Joins("JOIN lease_agreements ON lease_agreements.id = payments.lease_id").
		Joins("JOIN properties ON properties.id = lease_agreements.property_id").
		Where("properties.owner_id = ? AND payments.status = ?", ownerID, models.PaymentStatusSuccess).
		Group("TO_CHAR(payments.payment_date, 'Mon')").
		Order("MIN(payments.payment_date)").
		Find(&results).Error

	return results, err
}

func (r *paymentRepository) GetByTenantID(tenantID string) ([]models.Payment, error) {
	var payments []models.Payment
	err := r.db.Table("payments").
		Select("payments.*").
		Joins("JOIN lease_agreements ON lease_agreements.id = payments.lease_id").
		Where("lease_agreements.tenant_id = ?", tenantID).
		Order("payments.payment_date DESC").
		Find(&payments).Error
	return payments, err
}
