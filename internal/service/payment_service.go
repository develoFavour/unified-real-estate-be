package service

import (
	"real-estate-backend/internal/repository"
)

type PaymentService interface {
	GetOwnerIncomeReport(ownerID string) (any, error)
}

type paymentService struct {
	repo repository.PaymentRepository
}

func NewPaymentService(repo repository.PaymentRepository) PaymentService {
	return &paymentService{repo}
}

func (s *paymentService) GetOwnerIncomeReport(ownerID string) (any, error) {
	payments, err := s.repo.GetByOwnerID(ownerID)
	if err != nil {
		return nil, err
	}

	monthly, err := s.repo.GetMonthlyIncome(ownerID)
	if err != nil {
		return nil, err
	}

	var totalRevenue float64
	for _, p := range payments {
		totalRevenue += p.Amount
	}

	return map[string]interface{}{
		"history":       payments,
		"monthly_stats": monthly,
		"total_revenue": totalRevenue,
	}, nil
}
