package service

import (
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/repository"

	"gorm.io/gorm"
)

type TenantService interface {
	GetTenantDashboard(tenantID string) (any, error)
	GetTenantPayments(tenantID string) ([]models.Payment, error)
}

type tenantService struct {
	leaseRepo repository.LeaseRepository
	propRepo  repository.PropertyRepository
	maintRepo repository.MaintenanceRepository
	payRepo   repository.PaymentRepository
	db        *gorm.DB
}

func NewTenantService(
	leaseRepo repository.LeaseRepository,
	propRepo repository.PropertyRepository,
	maintRepo repository.MaintenanceRepository,
	payRepo repository.PaymentRepository,
	db *gorm.DB,
) TenantService {
	return &tenantService{leaseRepo, propRepo, maintRepo, payRepo, db}
}

func (s *tenantService) GetTenantDashboard(tenantID string) (any, error) {
	// 1. Get Active Lease
	lease, err := s.leaseRepo.GetByTenantID(tenantID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			var pendingLease models.LeaseAgreement
			if pendingErr := s.db.Where("tenant_id = ? AND status = ?", tenantID, models.LeaseStatusPending).
				Order("created_at DESC").
				First(&pendingLease).Error; pendingErr != nil {
				if pendingErr == gorm.ErrRecordNotFound {
					return nil, nil
				}
				return nil, pendingErr
			}
			lease = &pendingLease
		} else {
			return nil, err
		}
	}

	// 2. Get Property Details
	prop, err := s.propRepo.GetPropertyByID(lease.PropertyID.String())
	if err != nil {
		return nil, err
	}

	// 3. Get Recent Maintenance
	maint, _ := s.maintRepo.GetByTenantID(tenantID)

	// 4. Get Recent Payments
	payments, _ := s.payRepo.GetByTenantID(tenantID)

	// 5. Get Wallet
	var wallet models.Wallet
	s.db.Where("user_id = ?", tenantID).First(&wallet)

	return map[string]interface{}{
		"lease":       lease,
		"property":    prop,
		"maintenance": maint,
		"payments":    payments,
		"wallet":      wallet,
	}, nil
}

func (s *tenantService) GetTenantPayments(tenantID string) ([]models.Payment, error) {
	return s.payRepo.GetByTenantID(tenantID)
}
