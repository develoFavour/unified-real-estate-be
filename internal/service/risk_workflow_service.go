package service

import (
	"encoding/json"
	"errors"
	"log"
	"real-estate-backend/internal/models"
	"real-estate-backend/pkg/mail"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RiskWorkflowService interface {
	GetMySaleReservations(userID string) ([]models.SaleReservation, error)
	GetIncomingSaleReservations(userID string) ([]models.SaleReservation, error)
	GetDocuments(userID string, input GetDocumentsInput) ([]models.TransactionDocument, error)
	UploadDocument(userID string, input UploadDocumentInput) (*models.TransactionDocument, error)
	AcceptLeaseDocuments(userID, leaseID string) (*models.LeaseAgreement, error)
	AcceptSaleDocuments(userID, reservationID string) (*models.SaleReservation, error)
	RecordFinalSettlement(userID, reservationID string, input FinalSettlementInput) (*models.SaleReservation, error)
	OpenDispute(userID string, input OpenDisputeInput) (*models.Dispute, error)
	GetMyDisputes(userID string) ([]models.Dispute, error)
	RespondToDispute(userID, disputeID string, response string) (*models.Dispute, error)
	ResolveDispute(userID, role, disputeID string, input ResolveDisputeInput) (*models.Dispute, error)
	RunReservationExpiryCheck() (int, error)
}

type riskWorkflowService struct {
	db          *gorm.DB
	mailService mail.MailService
}

type UploadDocumentInput struct {
	PropertyID        string `json:"property_id"`
	LeaseID           string `json:"lease_id"`
	InvoiceID         string `json:"invoice_id"`
	SaleReservationID string `json:"sale_reservation_id"`
	Type              string `json:"type"`
	Title             string `json:"title"`
	DocumentURL       string `json:"document_url"`
	Notes             string `json:"notes"`
}

type GetDocumentsInput struct {
	LeaseID           string
	InvoiceID         string
	SaleReservationID string
}

type FinalSettlementInput struct {
	Amount    float64    `json:"amount"`
	Reference string     `json:"reference"`
	Date      *time.Time `json:"date"`
	Notes     string     `json:"notes"`
}

type OpenDisputeInput struct {
	PropertyID          string `json:"property_id"`
	InvoiceID           string `json:"invoice_id"`
	SaleReservationID   string `json:"sale_reservation_id"`
	WalletTransactionID string `json:"wallet_transaction_id"`
	Reason              string `json:"reason"`
	Description         string `json:"description"`
}

type ResolveDisputeInput struct {
	Action     string `json:"action"`
	Status     string `json:"status"`
	Resolution string `json:"resolution"`
}

func NewRiskWorkflowService(db *gorm.DB, mailService mail.MailService) RiskWorkflowService {
	return &riskWorkflowService{db: db, mailService: mailService}
}

func (s *riskWorkflowService) GetMySaleReservations(userID string) ([]models.SaleReservation, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, errors.New("invalid user ID")
	}
	var reservations []models.SaleReservation
	err := s.db.Preload("Property").Preload("Documents").
		Where("buyer_id = ?", userID).
		Order("created_at DESC").
		Find(&reservations).Error
	return reservations, err
}

func (s *riskWorkflowService) GetIncomingSaleReservations(userID string) ([]models.SaleReservation, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, errors.New("invalid user ID")
	}
	var reservations []models.SaleReservation
	err := s.db.Preload("Property").Preload("Buyer").Preload("Buyer.Profile").Preload("Documents").
		Joins("JOIN properties ON properties.id = sale_reservations.property_id").
		Where("properties.owner_id = ? OR properties.agent_id = ?", userID, userID).
		Order("sale_reservations.created_at DESC").
		Find(&reservations).Error
	return reservations, err
}

func (s *riskWorkflowService) GetDocuments(userID string, input GetDocumentsInput) ([]models.TransactionDocument, error) {
	actorID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	query := s.db.Preload("UploadedBy").Preload("UploadedBy.Profile").Order("created_at DESC")
	if input.SaleReservationID != "" {
		var reservation models.SaleReservation
		if err := s.db.Preload("Property").Where("id = ?", input.SaleReservationID).First(&reservation).Error; err != nil {
			return nil, err
		}
		if reservation.BuyerID != actorID && (reservation.Property == nil || !s.isWorkflowStakeholder(*reservation.Property, actorID)) {
			return nil, errors.New("you cannot view documents for this reservation")
		}
		query = query.Where("sale_reservation_id = ?", reservation.ID)
	} else if input.LeaseID != "" {
		var lease models.LeaseAgreement
		if err := s.db.Where("id = ?", input.LeaseID).First(&lease).Error; err != nil {
			return nil, err
		}
		if lease.TenantID != actorID {
			if _, err := s.getPropertyForStakeholder(lease.PropertyID, actorID); err != nil {
				return nil, errors.New("you cannot view documents for this lease")
			}
		}
		query = query.Where("lease_id = ?", lease.ID)
	} else if input.InvoiceID != "" {
		invoice, err := s.getInvoiceVisibleToUser(input.InvoiceID, actorID)
		if err != nil {
			return nil, err
		}
		query = query.Where("invoice_id = ?", invoice.ID)
	} else {
		return nil, errors.New("document scope is required")
	}

	var documents []models.TransactionDocument
	if err := query.Find(&documents).Error; err != nil {
		return nil, err
	}
	return documents, nil
}

func (s *riskWorkflowService) UploadDocument(userID string, input UploadDocumentInput) (*models.TransactionDocument, error) {
	actorID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}
	docType, err := parseDocumentType(input.Type)
	if err != nil {
		return nil, err
	}
	if input.Title == "" || input.DocumentURL == "" {
		return nil, errors.New("document title and url are required")
	}

	document := &models.TransactionDocument{
		UploadedByID: actorID,
		Type:         docType,
		Title:        input.Title,
		DocumentURL:  input.DocumentURL,
		Notes:        input.Notes,
	}

	if input.SaleReservationID != "" {
		reservation, err := s.getReservationForStakeholder(input.SaleReservationID, actorID)
		if err != nil {
			return nil, err
		}
		document.SaleReservationID = &reservation.ID
		document.PropertyID = &reservation.PropertyID
	} else if input.LeaseID != "" {
		lease, err := s.getLeaseForStakeholder(input.LeaseID, actorID)
		if err != nil {
			return nil, err
		}
		document.LeaseID = &lease.ID
		document.PropertyID = &lease.PropertyID
	} else if input.InvoiceID != "" {
		invoice, err := s.getInvoiceForStakeholder(input.InvoiceID, actorID)
		if err != nil {
			return nil, err
		}
		document.InvoiceID = &invoice.ID
		document.PropertyID = &invoice.PropertyID
		if invoice.LeaseID != nil {
			document.LeaseID = invoice.LeaseID
		}
	} else if input.PropertyID != "" {
		propertyID, err := uuid.Parse(input.PropertyID)
		if err != nil {
			return nil, errors.New("invalid property ID")
		}
		property, err := s.getPropertyForStakeholder(propertyID, actorID)
		if err != nil {
			return nil, err
		}
		document.PropertyID = &property.ID
	} else {
		return nil, errors.New("document must be linked to a reservation, lease, invoice, or property")
	}

	if err := s.db.Create(document).Error; err != nil {
		return nil, err
	}
	s.audit(&actorID, "TRANSACTION_DOCUMENT", document.ID, "DOCUMENT_UPLOADED", map[string]any{
		"type":  document.Type,
		"title": document.Title,
	})
	if document.LeaseID != nil {
		s.notifyLeaseDocumentsReadyIfComplete(*document.LeaseID)
	}
	return document, nil
}

func (s *riskWorkflowService) AcceptLeaseDocuments(userID, leaseID string) (*models.LeaseAgreement, error) {
	tenantID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}
	var lease models.LeaseAgreement
	if err := s.db.Where("id = ?", leaseID).First(&lease).Error; err != nil {
		return nil, err
	}
	if lease.TenantID != tenantID {
		return nil, errors.New("only the tenant can acknowledge this lease")
	}
	var docsCount int64
	if err := s.db.Model(&models.TransactionDocument{}).
		Where("lease_id = ?", lease.ID).
		Count(&docsCount).Error; err != nil {
		return nil, err
	}
	if docsCount == 0 {
		return nil, errors.New("lease documents must be uploaded before tenant acknowledgement")
	}
	lease.TenantAccepted = true
	lease.Status = models.LeaseStatusActive
	if err := s.db.Save(&lease).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&models.Property{}).
		Where("id = ?", lease.PropertyID).
		Update("status", models.StatusRented).Error; err != nil {
		return nil, err
	}
	s.notifyLeaseActivated(lease)
	s.audit(&tenantID, "LEASE", lease.ID, "TENANT_ACKNOWLEDGED_DOCUMENTS", nil)
	return &lease, nil
}

func (s *riskWorkflowService) notifyLeaseActivated(lease models.LeaseAgreement) {
	var tenant models.User
	if err := s.db.Preload("Profile").Where("id = ?", lease.TenantID).First(&tenant).Error; err != nil {
		log.Printf("Failed to load tenant for lease activation email: %v", err)
		return
	}

	var property models.Property
	if err := s.db.Where("id = ?", lease.PropertyID).First(&property).Error; err != nil {
		log.Printf("Failed to load property for lease activation email: %v", err)
		return
	}

	s.notify(lease.TenantID, "Lease activated", "Your lease for "+property.Title+" is now active and the property has been marked rented.")

	if s.mailService == nil || tenant.Email == "" {
		return
	}
	name := tenant.Profile.FullName
	if name == "" {
		name = "there"
	}
	propertyTitle := property.Title
	if propertyTitle == "" {
		propertyTitle = "your property"
	}

	go func() {
		if err := s.mailService.SendLeaseActivatedEmail(tenant.Email, name, propertyTitle); err != nil {
			log.Printf("Failed to send lease activation email: %v", err)
		}
	}()
}

func (s *riskWorkflowService) notifyLeaseDocumentsReadyIfComplete(leaseID uuid.UUID) {
	requiredTypes := []models.TransactionDocumentType{
		models.DocumentLeaseAgreement,
		models.DocumentInspection,
		models.DocumentHandoverNote,
		models.DocumentIDVerification,
	}

	var uploadedTypesCount int64
	if err := s.db.Model(&models.TransactionDocument{}).
		Where("lease_id = ? AND type IN ?", leaseID, requiredTypes).
		Distinct("type").
		Count(&uploadedTypesCount).Error; err != nil {
		log.Printf("Failed to count lease documents for ready email: %v", err)
		return
	}
	if uploadedTypesCount < int64(len(requiredTypes)) {
		return
	}

	var existingNoticeCount int64
	if err := s.db.Model(&models.AuditEvent{}).
		Where("entity_type = ? AND entity_id = ? AND action = ?", "LEASE", leaseID, "LEASE_DOCUMENTS_READY_EMAIL_SENT").
		Count(&existingNoticeCount).Error; err != nil {
		log.Printf("Failed to check lease document ready email audit: %v", err)
		return
	}
	if existingNoticeCount > 0 {
		return
	}

	var lease models.LeaseAgreement
	if err := s.db.Where("id = ?", leaseID).First(&lease).Error; err != nil {
		log.Printf("Failed to load lease for document ready email: %v", err)
		return
	}
	if lease.TenantAccepted {
		return
	}

	var tenant models.User
	if err := s.db.Preload("Profile").Where("id = ?", lease.TenantID).First(&tenant).Error; err != nil {
		log.Printf("Failed to load tenant for document ready email: %v", err)
		return
	}

	var property models.Property
	if err := s.db.Where("id = ?", lease.PropertyID).First(&property).Error; err != nil {
		log.Printf("Failed to load property for document ready email: %v", err)
		return
	}

	propertyTitle := property.Title
	if propertyTitle == "" {
		propertyTitle = "your property"
	}
	s.notify(lease.TenantID, "Lease documents ready", "Documents for "+propertyTitle+" are ready for your review and acknowledgement.")

	s.audit(nil, "LEASE", lease.ID, "LEASE_DOCUMENTS_READY_EMAIL_SENT", map[string]any{
		"property_title": propertyTitle,
	})

	if s.mailService == nil || tenant.Email == "" {
		return
	}
	name := tenant.Profile.FullName
	if name == "" {
		name = "there"
	}

	go func() {
		if err := s.mailService.SendLeaseDocumentsReadyEmail(tenant.Email, name, propertyTitle); err != nil {
			log.Printf("Failed to send lease documents ready email: %v", err)
		}
	}()
}

func (s *riskWorkflowService) AcceptSaleDocuments(userID, reservationID string) (*models.SaleReservation, error) {
	buyerID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}
	var reservation models.SaleReservation
	if err := s.db.Preload("Documents").Where("id = ?", reservationID).First(&reservation).Error; err != nil {
		return nil, err
	}
	if reservation.BuyerID != buyerID {
		return nil, errors.New("only the buyer can accept sale documents")
	}
	if len(reservation.Documents) == 0 {
		return nil, errors.New("documents must be uploaded before buyer acceptance")
	}
	now := time.Now()
	reservation.BuyerAcceptedAt = &now
	if reservation.Status == models.SaleReservationReserved {
		reservation.Status = models.SaleReservationUnderReview
	}
	if err := s.db.Save(&reservation).Error; err != nil {
		return nil, err
	}
	s.audit(&buyerID, "SALE_RESERVATION", reservation.ID, "BUYER_ACCEPTED_DOCUMENTS", nil)
	return &reservation, nil
}

func (s *riskWorkflowService) RecordFinalSettlement(userID, reservationID string, input FinalSettlementInput) (*models.SaleReservation, error) {
	actorID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}
	if input.Amount <= 0 || input.Reference == "" {
		return nil, errors.New("settlement amount and reference are required")
	}

	returnReservation := models.SaleReservation{}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var reservation models.SaleReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Documents").
			Where("id = ?", reservationID).
			First(&reservation).Error; err != nil {
			return err
		}
		var property models.Property
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", reservation.PropertyID).First(&property).Error; err != nil {
			return err
		}
		if !s.isWorkflowStakeholder(property, actorID) {
			return errors.New("you can only record settlement for properties you own or manage")
		}
		if reservation.BuyerAcceptedAt == nil {
			return errors.New("buyer must review and accept sale documents before final settlement")
		}
		if len(reservation.Documents) == 0 {
			return errors.New("required sale documents must be uploaded before final settlement")
		}
		var openDisputeCount int64
		if err := tx.Model(&models.Dispute{}).
			Where("sale_reservation_id = ? AND status IN ?", reservation.ID, []models.DisputeStatus{models.DisputeOpen, models.DisputeResponded, models.DisputeUnderReview}).
			Count(&openDisputeCount).Error; err != nil {
			return err
		}
		if openDisputeCount > 0 {
			return errors.New("open disputes must be resolved before marking sold")
		}

		settlementDate := time.Now()
		if input.Date != nil {
			settlementDate = *input.Date
		}
		reservation.FinalSettlementAmount = input.Amount
		reservation.FinalSettlementReference = input.Reference
		reservation.FinalSettlementDate = &settlementDate
		reservation.FinalSettlementNotes = input.Notes
		reservation.Status = models.SaleReservationSold
		if err := tx.Save(&reservation).Error; err != nil {
			return err
		}
		if err := tx.Model(&property).Update("status", models.StatusSold).Error; err != nil {
			return err
		}
		if err := s.transferPropertyOwnership(tx, property, reservation); err != nil {
			return err
		}
		returnReservation = reservation
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.notify(returnReservation.BuyerID, "Sale confirmed", "Final settlement has been recorded and the property has been marked sold.")
	s.audit(&actorID, "SALE_RESERVATION", returnReservation.ID, "FINAL_SETTLEMENT_RECORDED", map[string]any{
		"amount":    returnReservation.FinalSettlementAmount,
		"reference": returnReservation.FinalSettlementReference,
	})
	return &returnReservation, nil
}

func (s *riskWorkflowService) OpenDispute(userID string, input OpenDisputeInput) (*models.Dispute, error) {
	openedByID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}
	if input.Reason == "" || input.Description == "" {
		return nil, errors.New("reason and description are required")
	}

	dispute := &models.Dispute{
		OpenedByID:  openedByID,
		Reason:      input.Reason,
		Description: input.Description,
		Status:      models.DisputeOpen,
	}

	if input.SaleReservationID != "" {
		var reservation models.SaleReservation
		if err := s.db.Preload("Property").Where("id = ?", input.SaleReservationID).First(&reservation).Error; err != nil {
			return nil, err
		}
		if reservation.BuyerID != openedByID && !s.isWorkflowStakeholder(*reservation.Property, openedByID) {
			return nil, errors.New("you cannot open a dispute for this reservation")
		}
		dispute.SaleReservationID = &reservation.ID
		dispute.PropertyID = &reservation.PropertyID
		dispute.RespondentID = reservation.Property.OwnerID
		reservation.Status = models.SaleReservationDisputed
		_ = s.db.Save(&reservation).Error
	} else if input.InvoiceID != "" {
		invoice, err := s.getInvoiceVisibleToUser(input.InvoiceID, openedByID)
		if err != nil {
			return nil, err
		}
		dispute.InvoiceID = &invoice.ID
		dispute.PropertyID = &invoice.PropertyID
		if invoice.Property != nil {
			dispute.RespondentID = invoice.Property.OwnerID
		}
	} else if input.WalletTransactionID != "" {
		txID, err := uuid.Parse(input.WalletTransactionID)
		if err != nil {
			return nil, errors.New("invalid wallet transaction ID")
		}
		dispute.WalletTransactionID = &txID
	} else if input.PropertyID != "" {
		propertyID, err := uuid.Parse(input.PropertyID)
		if err != nil {
			return nil, errors.New("invalid property ID")
		}
		dispute.PropertyID = &propertyID
	} else {
		return nil, errors.New("dispute must be linked to a reservation, invoice, transaction, or property")
	}

	if err := s.db.Create(dispute).Error; err != nil {
		return nil, err
	}
	if dispute.RespondentID != nil {
		s.notify(*dispute.RespondentID, "Dispute opened", "A dispute has been opened and needs your response.")
	}
	s.audit(&openedByID, "DISPUTE", dispute.ID, "DISPUTE_OPENED", map[string]any{"reason": dispute.Reason})
	return dispute, nil
}

func (s *riskWorkflowService) GetMyDisputes(userID string) ([]models.Dispute, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, errors.New("invalid user ID")
	}
	var disputes []models.Dispute
	err := s.db.Preload("Property").Preload("Invoice").Preload("SaleReservation").
		Where("opened_by_id = ? OR respondent_id = ?", userID, userID).
		Order("created_at DESC").
		Find(&disputes).Error
	return disputes, err
}

func (s *riskWorkflowService) RespondToDispute(userID, disputeID string, response string) (*models.Dispute, error) {
	actorID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}
	if response == "" {
		return nil, errors.New("response is required")
	}
	var dispute models.Dispute
	if err := s.db.Preload("Property").Where("id = ?", disputeID).First(&dispute).Error; err != nil {
		return nil, err
	}
	if dispute.RespondentID != nil && *dispute.RespondentID != actorID {
		return nil, errors.New("you cannot respond to this dispute")
	}
	if dispute.Property != nil && !s.isWorkflowStakeholder(*dispute.Property, actorID) && dispute.RespondentID == nil {
		return nil, errors.New("you cannot respond to this dispute")
	}
	dispute.Response = response
	dispute.Status = models.DisputeResponded
	if err := s.db.Save(&dispute).Error; err != nil {
		return nil, err
	}
	s.notify(dispute.OpenedByID, "Dispute updated", "The other party has responded to your dispute.")
	s.audit(&actorID, "DISPUTE", dispute.ID, "DISPUTE_RESPONDED", nil)
	return &dispute, nil
}

func (s *riskWorkflowService) ResolveDispute(userID, role, disputeID string, input ResolveDisputeInput) (*models.Dispute, error) {
	actorID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}
	if models.Role(role) != models.RoleSuperAdmin {
		return nil, errors.New("only platform admins can resolve disputes")
	}
	if input.Resolution == "" {
		return nil, errors.New("resolution notes are required")
	}

	var resolvedDispute models.Dispute
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var dispute models.Dispute
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", disputeID).
			First(&dispute).Error; err != nil {
			return err
		}

		action := input.Action
		if action == "" {
			action = input.Status
		}
		status := models.DisputeResolved
		if action == string(models.DisputeRejected) || input.Status == string(models.DisputeRejected) || action == "REJECT_DISPUTE" {
			status = models.DisputeRejected
		}

		if dispute.SaleReservationID != nil {
			if err := s.applyDisputeResolutionToReservation(tx, *dispute.SaleReservationID, action); err != nil {
				return err
			}
		}

		dispute.Status = status
		dispute.Resolution = input.Resolution
		if err := tx.Save(&dispute).Error; err != nil {
			return err
		}
		resolvedDispute = dispute
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.notify(resolvedDispute.OpenedByID, "Dispute resolved", input.Resolution)
	if resolvedDispute.RespondentID != nil {
		s.notify(*resolvedDispute.RespondentID, "Dispute resolved", input.Resolution)
	}
	s.audit(&actorID, "DISPUTE", resolvedDispute.ID, "DISPUTE_RESOLVED", map[string]any{
		"status": resolvedDispute.Status,
		"action": input.Action,
	})
	return &resolvedDispute, nil
}

func (s *riskWorkflowService) applyDisputeResolutionToReservation(tx *gorm.DB, reservationID uuid.UUID, action string) error {
	if action == "" || action == string(models.DisputeResolved) || action == "RESOLVED" || action == "REJECT_DISPUTE" || action == string(models.DisputeRejected) {
		return nil
	}

	var reservation models.SaleReservation
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", reservationID).
		First(&reservation).Error; err != nil {
		return err
	}

	switch action {
	case "APPROVE_REFUND":
		reservation.Status = models.SaleReservationRefunded
		if err := tx.Save(&reservation).Error; err != nil {
			return err
		}
		return tx.Model(&models.Property{}).Where("id = ?", reservation.PropertyID).Update("status", models.StatusAvailable).Error
	case "FORCE_CANCEL_RESERVATION":
		reservation.Status = models.SaleReservationCancelled
		if err := tx.Save(&reservation).Error; err != nil {
			return err
		}
		return tx.Model(&models.Property{}).Where("id = ?", reservation.PropertyID).Update("status", models.StatusAvailable).Error
	case "FORCE_UNDER_REVIEW":
		reservation.Status = models.SaleReservationUnderReview
		if err := tx.Save(&reservation).Error; err != nil {
			return err
		}
		return tx.Model(&models.Property{}).Where("id = ?", reservation.PropertyID).Update("status", models.StatusUnderReview).Error
	case "CONFIRM_SOLD":
		if reservation.BuyerAcceptedAt == nil || reservation.FinalSettlementReference == "" || reservation.FinalSettlementAmount <= 0 {
			return errors.New("buyer acceptance and final settlement are required before confirming sold")
		}
		var property models.Property
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", reservation.PropertyID).First(&property).Error; err != nil {
			return err
		}
		reservation.Status = models.SaleReservationSold
		if err := tx.Save(&reservation).Error; err != nil {
			return err
		}
		if err := tx.Model(&property).Update("status", models.StatusSold).Error; err != nil {
			return err
		}
		return s.transferPropertyOwnership(tx, property, reservation)
	default:
		return errors.New("unsupported dispute resolution action")
	}
}

func (s *riskWorkflowService) transferPropertyOwnership(tx *gorm.DB, property models.Property, reservation models.SaleReservation) error {
	now := time.Now()

	if err := tx.Model(&models.PropertyOwnership{}).
		Where("property_id = ? AND is_current = ?", property.ID, true).
		Updates(map[string]any{
			"is_current": false,
			"ended_at":   &now,
		}).Error; err != nil {
		return err
	}

	var ownershipCount int64
	if err := tx.Model(&models.PropertyOwnership{}).
		Where("property_id = ?", property.ID).
		Count(&ownershipCount).Error; err != nil {
		return err
	}
	if ownershipCount == 0 && property.OwnerID != nil {
		if err := tx.Create(&models.PropertyOwnership{
			PropertyID: property.ID,
			UserID:     *property.OwnerID,
			Source:     models.OwnershipSourceListingOwner,
			IsCurrent:  false,
			StartedAt:  property.CreatedAt,
			EndedAt:    &now,
			Notes:      "Original listing owner before sale completion.",
		}).Error; err != nil {
			return err
		}
	}

	var currentBuyerOwnership models.PropertyOwnership
	err := tx.Where(
		"property_id = ? AND user_id = ? AND sale_reservation_id = ?",
		property.ID,
		reservation.BuyerID,
		reservation.ID,
	).First(&currentBuyerOwnership).Error
	if err == nil {
		return tx.Model(&currentBuyerOwnership).Updates(map[string]any{
			"is_current": true,
			"ended_at":   nil,
			"started_at": now,
			"source":     models.OwnershipSourceSale,
		}).Error
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return tx.Create(&models.PropertyOwnership{
		PropertyID:        property.ID,
		UserID:            reservation.BuyerID,
		SaleReservationID: &reservation.ID,
		Source:            models.OwnershipSourceSale,
		IsCurrent:         true,
		StartedAt:         now,
		Notes:             "Ownership recorded after sale final settlement.",
	}).Error
}

func (s *riskWorkflowService) RunReservationExpiryCheck() (int, error) {
	now := time.Now()
	var reservations []models.SaleReservation
	if err := s.db.Where(
		"status IN ? AND expires_at IS NOT NULL AND expires_at < ?",
		[]models.SaleReservationStatus{models.SaleReservationReserved, models.SaleReservationUnderReview},
		now,
	).Find(&reservations).Error; err != nil {
		return 0, err
	}
	for _, reservation := range reservations {
		reservation.Status = models.SaleReservationExpired
		if err := s.db.Save(&reservation).Error; err != nil {
			return 0, err
		}
		s.notify(reservation.BuyerID, "Reservation expired", "Your sale reservation has expired and can now be disputed or reviewed.")
		s.audit(nil, "SALE_RESERVATION", reservation.ID, "RESERVATION_EXPIRED", nil)
	}
	return len(reservations), nil
}

func (s *riskWorkflowService) getPropertyForStakeholder(propertyID uuid.UUID, actorID uuid.UUID) (*models.Property, error) {
	var property models.Property
	if err := s.db.Where("id = ?", propertyID).First(&property).Error; err != nil {
		return nil, err
	}
	if !s.isWorkflowStakeholder(property, actorID) {
		return nil, errors.New("you can only upload documents for properties you own or manage")
	}
	return &property, nil
}

func (s *riskWorkflowService) getReservationForStakeholder(reservationID string, actorID uuid.UUID) (*models.SaleReservation, error) {
	var reservation models.SaleReservation
	if err := s.db.Preload("Property").Where("id = ?", reservationID).First(&reservation).Error; err != nil {
		return nil, err
	}
	if reservation.Property == nil || !s.isWorkflowStakeholder(*reservation.Property, actorID) {
		return nil, errors.New("you can only upload sale documents for properties you own or manage")
	}
	return &reservation, nil
}

func (s *riskWorkflowService) getLeaseForStakeholder(leaseID string, actorID uuid.UUID) (*models.LeaseAgreement, error) {
	var lease models.LeaseAgreement
	if err := s.db.Where("id = ?", leaseID).First(&lease).Error; err != nil {
		return nil, err
	}
	if _, err := s.getPropertyForStakeholder(lease.PropertyID, actorID); err != nil {
		return nil, err
	}
	return &lease, nil
}

func (s *riskWorkflowService) getInvoiceForStakeholder(invoiceID string, actorID uuid.UUID) (*models.Invoice, error) {
	var invoice models.Invoice
	if err := s.db.Preload("Property").Where("id = ?", invoiceID).First(&invoice).Error; err != nil {
		return nil, err
	}
	if invoice.Property == nil || !s.isWorkflowStakeholder(*invoice.Property, actorID) {
		return nil, errors.New("you can only upload invoice documents for properties you own or manage")
	}
	return &invoice, nil
}

func (s *riskWorkflowService) getInvoiceVisibleToUser(invoiceID string, actorID uuid.UUID) (*models.Invoice, error) {
	var invoice models.Invoice
	if err := s.db.Preload("Property").Where("id = ?", invoiceID).First(&invoice).Error; err != nil {
		return nil, err
	}
	if invoice.TenantID == actorID {
		return &invoice, nil
	}
	if invoice.Property != nil && s.isWorkflowStakeholder(*invoice.Property, actorID) {
		return &invoice, nil
	}
	return nil, errors.New("you cannot open a dispute for this invoice")
}

func (s *riskWorkflowService) audit(actorID *uuid.UUID, entityType string, entityID uuid.UUID, action string, details map[string]any) {
	if s.db == nil {
		return
	}
	payload := ""
	if details != nil {
		raw, _ := json.Marshal(details)
		payload = string(raw)
	}
	_ = s.db.Create(&models.AuditEvent{
		ActorID:    actorID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Details:    payload,
	}).Error
}

func (s *riskWorkflowService) notify(userID uuid.UUID, title, message string) {
	_ = s.db.Create(&models.Notification{
		UserID:  userID,
		Title:   title,
		Message: message,
	}).Error
}

func parseDocumentType(value string) (models.TransactionDocumentType, error) {
	switch models.TransactionDocumentType(value) {
	case models.DocumentLeaseAgreement,
		models.DocumentInspection,
		models.DocumentHandoverNote,
		models.DocumentIDVerification,
		models.DocumentTitle,
		models.DocumentOwnershipProof,
		models.DocumentSaleAgreement,
		models.DocumentLegal,
		models.DocumentOther:
		return models.TransactionDocumentType(value), nil
	default:
		return "", errors.New("unsupported document type")
	}
}

func (s *riskWorkflowService) isWorkflowStakeholder(property models.Property, userID uuid.UUID) bool {
	if property.OwnerID != nil && *property.OwnerID == userID {
		return true
	}
	if property.AgentID != nil && *property.AgentID == userID {
		return true
	}
	var ownershipCount int64
	if err := s.db.Model(&models.PropertyOwnership{}).
		Where("property_id = ? AND user_id = ? AND is_current = ?", property.ID, userID, true).
		Count(&ownershipCount).Error; err == nil && ownershipCount > 0 {
		return true
	}
	return false
}
