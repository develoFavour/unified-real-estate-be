package service

import (
	"errors"
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/repository"
)

type LeaseService interface {
	CreateLease(req *models.LeaseAgreement) error
	GetLeaseByID(id string) (*models.LeaseAgreement, error)
	AcceptLease(leaseID string, tenantID string) error
}

type leaseService struct {
	repo repository.LeaseRepository
}

func NewLeaseService(repo repository.LeaseRepository) LeaseService {
	return &leaseService{repo}
}

func (s *leaseService) CreateLease(req *models.LeaseAgreement) error {
	return s.repo.CreateLease(req)
}

func (s *leaseService) GetLeaseByID(id string) (*models.LeaseAgreement, error) {
	return s.repo.GetLeaseByID(id)
}

func (s *leaseService) AcceptLease(leaseID string, tenantID string) error {
	lease, err := s.repo.GetLeaseByID(leaseID)
	if err != nil {
		return err
	}

	if lease.TenantID.String() != tenantID {
		return errors.New("unauthorized to accept this lease")
	}

	lease.TenantAccepted = true
	lease.Status = models.LeaseStatusActive

	return s.repo.UpdateLease(lease)
}
