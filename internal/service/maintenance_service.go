package service

import (
	"errors"
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/repository"
	"real-estate-backend/pkg/mail"

	"github.com/google/uuid"
)

type MaintenanceService interface {
	GetOwnerRequests(ownerID string) ([]models.MaintenanceRequest, error)
	GetAgentRequests(agentID string) ([]models.MaintenanceRequest, error)
	CreateRequest(req *models.MaintenanceRequest) error
	UpdateStatus(id string, status models.MaintenanceStatus, note, actorID string) error
	CloseRequest(id, tenantID string) error
	ReopenRequest(id, tenantID string) error
}

type maintenanceService struct {
	repo      repository.MaintenanceRepository
	leaseRepo repository.LeaseRepository
	propRepo  repository.PropertyRepository
	userRepo  repository.UserRepository
	mailRepo  mail.MailService
}

func NewMaintenanceService(
	repo repository.MaintenanceRepository,
	leaseRepo repository.LeaseRepository,
	propRepo repository.PropertyRepository,
	userRepo repository.UserRepository,
	mailRepo mail.MailService,
) MaintenanceService {
	return &maintenanceService{repo: repo, leaseRepo: leaseRepo, propRepo: propRepo, userRepo: userRepo, mailRepo: mailRepo}
}

func (s *maintenanceService) GetOwnerRequests(ownerID string) ([]models.MaintenanceRequest, error) {
	return s.repo.GetByOwnerID(ownerID)
}

func (s *maintenanceService) GetAgentRequests(agentID string) ([]models.MaintenanceRequest, error) {
	return s.repo.GetByAgentID(agentID)
}

func (s *maintenanceService) CreateRequest(req *models.MaintenanceRequest) error {
	lease, err := s.leaseRepo.GetByTenantID(req.TenantID.String())
	if err != nil {
		return errors.New("you can only report maintenance for an active lease")
	}
	if lease.PropertyID != req.PropertyID {
		return errors.New("this property is not linked to your active lease")
	}
	req.Status = models.MaintenanceStatusPending
	if err := s.repo.CreateRequest(req); err != nil {
		return err
	}
	tenantID := req.TenantID
	_ = s.repo.AddUpdate(&models.MaintenanceUpdate{
		RequestID: req.ID,
		ActorID:   &tenantID,
		Status:    models.MaintenanceStatusPending,
		Note:      "Maintenance request submitted by tenant.",
	})
	go s.notifyRequestSubmitted(req)
	return nil
}

func (s *maintenanceService) UpdateStatus(id string, status models.MaintenanceStatus, note, actorID string) error {
	if !isStakeholderMaintenanceStatus(status) {
		return errors.New("unsupported maintenance status")
	}
	request, err := s.repo.GetRequestByID(id)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateStatus(id, status, note); err != nil {
		return err
	}
	actorUUID, _ := uuid.Parse(actorID)
	_ = s.repo.AddUpdate(&models.MaintenanceUpdate{
		RequestID: request.ID,
		ActorID:   &actorUUID,
		Status:    status,
		Note:      note,
	})
	request.Status = status
	request.StatusNote = note
	go s.notifyTenantStatusUpdated(request, status, note)
	return nil
}

func (s *maintenanceService) CloseRequest(id, tenantID string) error {
	request, err := s.repo.GetRequestByID(id)
	if err != nil {
		return err
	}
	if request.TenantID.String() != tenantID {
		return errors.New("you can only close your own maintenance request")
	}
	if request.Status != models.MaintenanceStatusResolved {
		return errors.New("only resolved requests can be closed")
	}
	if err := s.repo.UpdateStatus(id, models.MaintenanceStatusClosed, "Tenant confirmed the issue is fixed."); err != nil {
		return err
	}
	tenantUUID := request.TenantID
	if err := s.repo.AddUpdate(&models.MaintenanceUpdate{
		RequestID: request.ID,
		ActorID:   &tenantUUID,
		Status:    models.MaintenanceStatusClosed,
		Note:      "Tenant confirmed the issue is fixed.",
	}); err != nil {
		return err
	}
	request.Status = models.MaintenanceStatusClosed
	go s.notifyManagerStatusUpdated(request, models.MaintenanceStatusClosed, "Tenant confirmed the issue is fixed.")
	return nil
}

func (s *maintenanceService) ReopenRequest(id, tenantID string) error {
	request, err := s.repo.GetRequestByID(id)
	if err != nil {
		return err
	}
	if request.TenantID.String() != tenantID {
		return errors.New("you can only reopen your own maintenance request")
	}
	if request.Status != models.MaintenanceStatusResolved && request.Status != models.MaintenanceStatusClosed {
		return errors.New("only resolved or closed requests can be reopened")
	}
	if err := s.repo.UpdateStatus(id, models.MaintenanceStatusReopened, "Tenant reopened the request because the issue still needs attention."); err != nil {
		return err
	}
	tenantUUID := request.TenantID
	if err := s.repo.AddUpdate(&models.MaintenanceUpdate{
		RequestID: request.ID,
		ActorID:   &tenantUUID,
		Status:    models.MaintenanceStatusReopened,
		Note:      "Tenant reopened the request because the issue still needs attention.",
	}); err != nil {
		return err
	}
	request.Status = models.MaintenanceStatusReopened
	go s.notifyManagerStatusUpdated(request, models.MaintenanceStatusReopened, "Tenant reopened the request because the issue still needs attention.")
	return nil
}

func isStakeholderMaintenanceStatus(status models.MaintenanceStatus) bool {
	switch status {
	case models.MaintenanceStatusPending,
		models.MaintenanceStatusAcknowledged,
		models.MaintenanceStatusInProgress,
		models.MaintenanceStatusResolved:
		return true
	default:
		return false
	}
}

func (s *maintenanceService) notifyRequestSubmitted(req *models.MaintenanceRequest) {
	prop, err := s.propRepo.GetPropertyByID(req.PropertyID.String())
	if err != nil || prop == nil {
		return
	}

	recipient := prop.Owner
	if prop.Agent != nil && prop.AgentAssignmentStatus == "ACCEPTED" {
		recipient = prop.Agent
	}
	if recipient == nil {
		return
	}

	name := recipient.Profile.FullName
	if name == "" {
		name = recipient.Email
	}
	if err := s.mailRepo.SendMaintenanceRequestSubmittedEmail(recipient.Email, name, string(recipient.Role), prop.Title, req.Title, string(req.Priority)); err != nil {
		// Email failure should not block maintenance workflow.
		return
	}
}

func (s *maintenanceService) notifyTenantStatusUpdated(req *models.MaintenanceRequest, status models.MaintenanceStatus, note string) {
	tenant, err := s.userRepo.FindByID(req.TenantID.String())
	if err != nil || tenant == nil {
		return
	}
	propTitle := "your property"
	if req.Property != nil {
		propTitle = req.Property.Title
	} else if prop, err := s.propRepo.GetPropertyByID(req.PropertyID.String()); err == nil && prop != nil {
		propTitle = prop.Title
	}
	name := tenant.Profile.FullName
	if name == "" {
		name = tenant.Email
	}
	_ = s.mailRepo.SendMaintenanceStatusUpdatedEmail(tenant.Email, name, propTitle, req.Title, string(status), note)
}

func (s *maintenanceService) notifyManagerStatusUpdated(req *models.MaintenanceRequest, status models.MaintenanceStatus, note string) {
	prop, err := s.propRepo.GetPropertyByID(req.PropertyID.String())
	if err != nil || prop == nil {
		return
	}
	recipient := prop.Owner
	if prop.Agent != nil && prop.AgentAssignmentStatus == "ACCEPTED" {
		recipient = prop.Agent
	}
	if recipient == nil {
		return
	}
	name := recipient.Profile.FullName
	if name == "" {
		name = recipient.Email
	}
	_ = s.mailRepo.SendMaintenanceManagerStatusUpdatedEmail(recipient.Email, name, string(recipient.Role), prop.Title, req.Title, string(status), note)
}
