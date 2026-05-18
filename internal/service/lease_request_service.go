package service

import (
	"errors"
	"log"
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/repository"
	"real-estate-backend/pkg/mail"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LeaseRequestService interface {
	CreateRequest(tenantID, propertyID, message string) (*models.LeaseRequest, error)
	GetMyRequests(tenantID string) ([]models.LeaseRequest, error)
	GetIncomingRequests(userID string) ([]models.LeaseRequest, error)
	ReviewRequest(userID, requestID string, status models.LeaseRequestStatus) (*models.LeaseRequest, error)
}

type leaseRequestService struct {
	reqRepo        repository.LeaseRequestRepository
	propRepo       repository.PropertyRepository
	leaseRepo      repository.LeaseRepository
	invoiceService InvoiceService
	notifyRepo     repository.NotificationRepository
	mailService    mail.MailService
}

func NewLeaseRequestService(reqRepo repository.LeaseRequestRepository, propRepo repository.PropertyRepository, leaseRepo repository.LeaseRepository, invoiceService InvoiceService, notifyRepo repository.NotificationRepository, mailService mail.MailService) LeaseRequestService {
	return &leaseRequestService{reqRepo: reqRepo, propRepo: propRepo, leaseRepo: leaseRepo, invoiceService: invoiceService, notifyRepo: notifyRepo, mailService: mailService}
}

func (s *leaseRequestService) CreateRequest(tenantID, propertyID, message string) (*models.LeaseRequest, error) {
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, errors.New("invalid tenant ID")
	}
	propertyUUID, err := uuid.Parse(propertyID)
	if err != nil {
		return nil, errors.New("invalid property ID")
	}

	property, err := s.propRepo.GetPropertyByID(propertyID)
	if err != nil {
		return nil, errors.New("property not found")
	}
	if property.ListingType != models.TypeRent {
		return nil, errors.New("lease requests are only available for rental properties")
	}
	if property.Status != models.StatusAvailable {
		return nil, errors.New("this property is no longer available for lease requests")
	}

	activeLease, err := s.leaseRepo.GetByTenantID(tenantID)
	if err == nil && activeLease != nil && activeLease.PropertyID != propertyUUID {
		return nil, errors.New("you already have an active lease. End or terminate it before requesting another rental property")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	existing, err := s.reqRepo.GetActiveByTenantAndProperty(tenantUUID, propertyUUID)
	if err == nil && existing != nil {
		return nil, errors.New("you already have an active lease request for this property")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	req := &models.LeaseRequest{
		PropertyID: propertyUUID,
		TenantID:   tenantUUID,
		Message:    message,
		Status:     models.LeaseRequestPending,
	}
	if err := s.reqRepo.Create(req); err != nil {
		return nil, err
	}

	return s.reqRepo.GetByID(req.ID.String())
}

func (s *leaseRequestService) GetMyRequests(tenantID string) ([]models.LeaseRequest, error) {
	return s.reqRepo.GetByTenantID(tenantID)
}

func (s *leaseRequestService) GetIncomingRequests(userID string) ([]models.LeaseRequest, error) {
	return s.reqRepo.GetIncomingForReviewer(userID)
}

func (s *leaseRequestService) ReviewRequest(userID, requestID string, status models.LeaseRequestStatus) (*models.LeaseRequest, error) {
	if status != models.LeaseRequestApproved && status != models.LeaseRequestRejected {
		return nil, errors.New("status must be APPROVED or REJECTED")
	}

	req, err := s.reqRepo.GetByID(requestID)
	if err != nil {
		return nil, errors.New("lease request not found")
	}
	if req.Status != models.LeaseRequestPending {
		return nil, errors.New("lease request has already been reviewed")
	}
	if req.Property == nil {
		return nil, errors.New("lease request property not found")
	}

	isOwner := req.Property.OwnerID != nil && req.Property.OwnerID.String() == userID
	isAgent := req.Property.AgentID != nil && req.Property.AgentID.String() == userID
	if !isOwner && !isAgent {
		return nil, errors.New("you can only review lease requests for properties you own or manage")
	}

	reviewerID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid reviewer ID")
	}

	now := time.Now()
	req.Status = status
	req.ReviewerID = &reviewerID
	req.ReviewedAt = &now

	if err := s.reqRepo.Update(req); err != nil {
		return nil, err
	}

	updatedReq, err := s.reqRepo.GetByID(req.ID.String())
	if err != nil {
		return nil, err
	}

	if status == models.LeaseRequestApproved && s.invoiceService != nil {
		invoice, err := s.invoiceService.CreateRentInvoiceForLeaseRequest(updatedReq)
		if err != nil {
			return nil, err
		}
		s.notifyLeaseApproved(updatedReq, invoice)
	}

	return updatedReq, nil
}

func (s *leaseRequestService) notifyLeaseApproved(req *models.LeaseRequest, invoice *models.Invoice) {
	if req == nil || invoice == nil || req.Tenant == nil || req.Property == nil {
		return
	}

	propertyTitle := req.Property.Title
	if s.notifyRepo != nil {
		_ = s.notifyRepo.CreateNotification(&models.Notification{
			UserID:  req.TenantID,
			Title:   "Lease approved",
			Message: "Your lease request for " + propertyTitle + " was approved. A rent invoice is ready for payment.",
		})
	}

	if s.mailService == nil || req.Tenant.Email == "" {
		return
	}

	name := req.Tenant.Profile.FullName
	if name == "" {
		name = "there"
	}
	dueDate := "soon"
	if invoice.DueDate != nil {
		dueDate = invoice.DueDate.Format("Jan 2, 2006")
	}

	go func() {
		if err := s.mailService.SendLeaseApprovedEmail(req.Tenant.Email, name, propertyTitle, invoice.Amount, dueDate); err != nil {
			log.Printf("Failed to send lease approval email: %v", err)
		}
	}()
}
