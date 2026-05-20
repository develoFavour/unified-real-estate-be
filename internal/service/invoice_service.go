package service

import (
	"errors"
	"fmt"
	"log"
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/repository"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvoiceService interface {
	CreateRentInvoiceForLeaseRequest(req *models.LeaseRequest) (*models.Invoice, error)
	GetTenantInvoices(tenantID string) ([]models.Invoice, error)
	GetIncomingInvoices(userID string) ([]models.Invoice, error)
	CreateManualInvoice(requesterID string, input CreateManualInvoiceInput) (*models.Invoice, error)
	RunBillingCycle() (BillingCycleResult, error)
	StartRecurringBillingScheduler()
}

type invoiceService struct {
	repo       repository.InvoiceRepository
	notifyRepo repository.NotificationRepository
	db         *gorm.DB
}

type CreateManualInvoiceInput struct {
	TenantID    string     `json:"tenant_id"`
	PropertyID  string     `json:"property_id"`
	LeaseID     string     `json:"lease_id"`
	Type        string     `json:"type"`
	Amount      float64    `json:"amount"`
	DueDate     *time.Time `json:"due_date"`
	Description string     `json:"description"`
}

type BillingCycleResult struct {
	RenewalInvoicesCreated int `json:"renewal_invoices_created"`
	OverdueInvoicesMarked  int `json:"overdue_invoices_marked"`
	NotificationsSent      int `json:"notifications_sent"`
}

func NewInvoiceService(repo repository.InvoiceRepository, db *gorm.DB, notifyRepo repository.NotificationRepository) InvoiceService {
	return &invoiceService{repo: repo, db: db, notifyRepo: notifyRepo}
}

func (s *invoiceService) CreateRentInvoiceForLeaseRequest(req *models.LeaseRequest) (*models.Invoice, error) {
	if req == nil || req.Property == nil {
		return nil, errors.New("lease request property is required")
	}

	var existing models.Invoice
	err := s.db.Where(
		"lease_request_id = ? AND type = ? AND status IN ?",
		req.ID,
		models.InvoiceRent,
		[]models.InvoiceStatus{models.InvoicePending, models.InvoiceOverdue, models.InvoicePaid},
	).Order("created_at DESC").First(&existing).Error
	if err == nil {
		return s.repo.GetByID(existing.ID.String())
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	err = s.db.Where(
		"tenant_id = ? AND property_id = ? AND type = ? AND status IN ? AND billing_period_start IS NULL",
		req.TenantID,
		req.PropertyID,
		models.InvoiceRent,
		[]models.InvoiceStatus{models.InvoicePending, models.InvoiceOverdue, models.InvoicePaid},
	).Order("created_at DESC").First(&existing).Error
	if err == nil {
		return s.repo.GetByID(existing.ID.String())
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	dueDate := time.Now().AddDate(0, 0, 7)
	invoice := &models.Invoice{
		TenantID:       req.TenantID,
		PropertyID:     req.PropertyID,
		LeaseRequestID: &req.ID,
		Type:           models.InvoiceRent,
		Amount:         req.Property.Price,
		Status:         models.InvoicePending,
		DueDate:        &dueDate,
		Description:    "Initial rent invoice for approved lease request",
	}
	if err := s.repo.Create(invoice); err != nil {
		return nil, err
	}

	return s.repo.GetByID(invoice.ID.String())
}

func (s *invoiceService) GetTenantInvoices(tenantID string) ([]models.Invoice, error) {
	if _, err := uuid.Parse(tenantID); err != nil {
		return nil, errors.New("invalid tenant ID")
	}
	return s.repo.GetByTenantID(tenantID)
}

func (s *invoiceService) GetIncomingInvoices(userID string) ([]models.Invoice, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, errors.New("invalid user ID")
	}
	return s.repo.GetByPropertyStakeholder(userID)
}

func (s *invoiceService) CreateManualInvoice(requesterID string, input CreateManualInvoiceInput) (*models.Invoice, error) {
	if s.db == nil {
		return nil, errors.New("database is not available")
	}
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, errors.New("invalid requester ID")
	}
	if input.Amount <= 0 {
		return nil, errors.New("invoice amount must be greater than zero")
	}

	invoiceType, err := parseInvoiceType(input.Type)
	if err != nil {
		return nil, err
	}

	var lease models.LeaseAgreement
	if input.LeaseID != "" {
		if err := s.db.Where("id = ?", input.LeaseID).First(&lease).Error; err != nil {
			return nil, err
		}
	} else {
		if input.TenantID == "" || input.PropertyID == "" {
			return nil, errors.New("tenant_id and property_id are required when lease_id is not provided")
		}
		if _, err := uuid.Parse(input.TenantID); err != nil {
			return nil, errors.New("invalid tenant ID")
		}
		if _, err := uuid.Parse(input.PropertyID); err != nil {
			return nil, errors.New("invalid property ID")
		}
		if err := s.db.Where("tenant_id = ? AND property_id = ? AND status = ?", input.TenantID, input.PropertyID, models.LeaseStatusActive).
			First(&lease).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	var property models.Property
	propertyID := input.PropertyID
	tenantID := input.TenantID
	var leaseID *uuid.UUID
	if lease.ID != uuid.Nil {
		propertyID = lease.PropertyID.String()
		tenantID = lease.TenantID.String()
		leaseID = &lease.ID
	}
	if propertyID == "" || tenantID == "" {
		return nil, errors.New("tenant and property are required")
	}

	if err := s.db.Where("id = ?", propertyID).First(&property).Error; err != nil {
		return nil, err
	}
	if !isPropertyStakeholder(property, requesterUUID) {
		return nil, errors.New("you can only issue invoices for properties you own or manage")
	}

	tenantUUID, _ := uuid.Parse(tenantID)
	propertyUUID, _ := uuid.Parse(propertyID)
	if err := s.validateManualInvoiceAllowed(invoiceType, tenantUUID, propertyUUID, leaseID); err != nil {
		return nil, err
	}

	dueDate := time.Now().AddDate(0, 0, 7)
	if input.DueDate != nil {
		dueDate = *input.DueDate
	}
	description := input.Description
	if description == "" {
		description = fmt.Sprintf("%s invoice for %s", invoiceType, property.Title)
	}

	invoice := &models.Invoice{
		TenantID:    tenantUUID,
		PropertyID:  propertyUUID,
		LeaseID:     leaseID,
		Type:        invoiceType,
		Amount:      input.Amount,
		Status:      models.InvoicePending,
		DueDate:     &dueDate,
		Description: description,
	}
	if err := s.repo.Create(invoice); err != nil {
		return nil, err
	}

	s.sendNotification(tenantUUID, "New invoice issued", fmt.Sprintf("%s for %s is due on %s.", invoiceType, property.Title, dueDate.Format("Jan 2, 2006")))
	return s.repo.GetByID(invoice.ID.String())
}

func (s *invoiceService) validateManualInvoiceAllowed(invoiceType models.InvoiceType, tenantID, propertyID uuid.UUID, leaseID *uuid.UUID) error {
	if invoiceType == models.InvoiceRent {
		return errors.New("rent invoices are created automatically after lease approval or by the renewal billing cycle")
	}

	if !isMoveInInvoiceType(invoiceType) {
		return nil
	}

	query := s.db.Model(&models.Invoice{}).
		Where("tenant_id = ? AND property_id = ? AND type = ? AND status IN ?",
			tenantID,
			propertyID,
			invoiceType,
			[]models.InvoiceStatus{models.InvoicePending, models.InvoiceOverdue, models.InvoicePaid},
		)
	if leaseID != nil {
		query = query.Where("lease_id = ? OR lease_id IS NULL", *leaseID)
	}

	var existingCount int64
	if err := query.Count(&existingCount).Error; err != nil {
		return err
	}
	if existingCount > 0 {
		return fmt.Errorf("%s invoice already exists for this tenant and property", invoiceType)
	}

	return nil
}

func isMoveInInvoiceType(invoiceType models.InvoiceType) bool {
	switch invoiceType {
	case models.InvoiceCautionDeposit,
		models.InvoiceAgencyFee,
		models.InvoiceLegalFee:
		return true
	default:
		return false
	}
}

func (s *invoiceService) RunBillingCycle() (BillingCycleResult, error) {
	if s.db == nil {
		return BillingCycleResult{}, errors.New("database is not available")
	}

	now := time.Now()
	result := BillingCycleResult{}
	renewalWindowEnd := now.AddDate(0, 0, 30)

	var leases []models.LeaseAgreement
	if err := s.db.Where("status = ? AND end_date <= ?", models.LeaseStatusActive, renewalWindowEnd).
		Find(&leases).Error; err != nil {
		return result, err
	}

	for _, lease := range leases {
		created, notified, err := s.ensureRenewalInvoice(lease)
		if err != nil {
			return result, err
		}
		if created {
			result.RenewalInvoicesCreated++
		}
		if notified {
			result.NotificationsSent++
		}
	}

	var overdueInvoices []models.Invoice
	if err := s.db.Where("status = ? AND due_date IS NOT NULL AND due_date < ?", models.InvoicePending, now).
		Find(&overdueInvoices).Error; err != nil {
		return result, err
	}

	for _, invoice := range overdueInvoices {
		invoice.Status = models.InvoiceOverdue
		if err := s.db.Save(&invoice).Error; err != nil {
			return result, err
		}
		result.OverdueInvoicesMarked++
		if s.sendNotification(invoice.TenantID, "Invoice overdue", fmt.Sprintf("Your %s invoice is overdue. Please fund your wallet and complete payment.", invoice.Type)) {
			result.NotificationsSent++
		}
	}

	return result, nil
}

func (s *invoiceService) StartRecurringBillingScheduler() {
	if s.db == nil {
		return
	}
	go func() {
		if result, err := s.RunBillingCycle(); err != nil {
			log.Printf("Recurring billing cycle failed: %v", err)
		} else {
			log.Printf("Recurring billing cycle completed: %+v", result)
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if result, err := s.RunBillingCycle(); err != nil {
				log.Printf("Recurring billing cycle failed: %v", err)
			} else {
				log.Printf("Recurring billing cycle completed: %+v", result)
			}
		}
	}()
}

func (s *invoiceService) ensureRenewalInvoice(lease models.LeaseAgreement) (bool, bool, error) {
	periodStart := lease.EndDate
	periodEnd := lease.EndDate.AddDate(1, 0, 0)

	var existing models.Invoice
	err := s.db.Where(
		"lease_id = ? AND type = ? AND billing_period_start = ? AND status IN ?",
		lease.ID,
		models.InvoiceRent,
		periodStart,
		[]models.InvoiceStatus{models.InvoicePending, models.InvoicePaid, models.InvoiceOverdue},
	).First(&existing).Error
	if err == nil {
		return false, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, false, err
	}

	var property models.Property
	if err := s.db.Where("id = ?", lease.PropertyID).First(&property).Error; err != nil {
		return false, false, err
	}

	invoice := &models.Invoice{
		TenantID:           lease.TenantID,
		PropertyID:         lease.PropertyID,
		LeaseID:            &lease.ID,
		Type:               models.InvoiceRent,
		Amount:             lease.RentAmount,
		Status:             models.InvoicePending,
		DueDate:            &lease.EndDate,
		BillingPeriodStart: &periodStart,
		BillingPeriodEnd:   &periodEnd,
		Description:        fmt.Sprintf("Renewal rent invoice for %s", property.Title),
	}
	if err := s.repo.Create(invoice); err != nil {
		return false, false, err
	}

	notified := s.sendNotification(lease.TenantID, "Rent renewal invoice ready", fmt.Sprintf("Your renewal rent invoice for %s is due on %s.", property.Title, lease.EndDate.Format("Jan 2, 2006")))
	return true, notified, nil
}

func (s *invoiceService) sendNotification(userID uuid.UUID, title, message string) bool {
	if s.notifyRepo == nil {
		return false
	}
	err := s.notifyRepo.CreateNotification(&models.Notification{
		UserID:  userID,
		Title:   title,
		Message: message,
	})
	if err != nil {
		log.Printf("Failed to create notification: %v", err)
		return false
	}
	return true
}

func parseInvoiceType(value string) (models.InvoiceType, error) {
	switch models.InvoiceType(value) {
	case models.InvoiceRent,
		models.InvoiceCautionDeposit,
		models.InvoiceAgencyFee,
		models.InvoiceLegalFee,
		models.InvoiceServiceCharge,
		models.InvoiceReservationDeposit:
		return models.InvoiceType(value), nil
	default:
		return "", errors.New("unsupported invoice type")
	}
}

func isPropertyStakeholder(property models.Property, userID uuid.UUID) bool {
	if property.OwnerID != nil && *property.OwnerID == userID {
		return true
	}
	if property.AgentID != nil && *property.AgentID == userID {
		return true
	}
	return false
}
