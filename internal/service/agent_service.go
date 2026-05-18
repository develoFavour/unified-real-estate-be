package service

import (
	"errors"
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/repository"
	"github.com/google/uuid"
)

type AgentService interface {
	GetAgentMandates(agentID string) ([]models.Property, error)
	GetAgentInvitations(email string) ([]models.Invitation, error)
	AcceptInvitation(invitationID, agentID string) error
	AcceptMandate(propertyID, agentID string) error
	GetAgentDashboardSummary(agentID, email string) (any, error)
}

type agentService struct {
	propRepo   repository.PropertyRepository
	inviteRepo repository.InvitationRepository
	maintRepo  repository.MaintenanceRepository
}

func NewAgentService(
	propRepo repository.PropertyRepository,
	inviteRepo repository.InvitationRepository,
	maintRepo repository.MaintenanceRepository,
) AgentService {
	return &agentService{propRepo, inviteRepo, maintRepo}
}

func (s *agentService) GetAgentMandates(agentID string) ([]models.Property, error) {
	// Find properties where AgentID matches
	return s.propRepo.GetByAgentID(agentID)
}

func (s *agentService) GetAgentInvitations(email string) ([]models.Invitation, error) {
	return s.inviteRepo.FindByEmail(email)
}

func (s *agentService) AcceptInvitation(invitationID, agentID string) error {
	invite, err := s.inviteRepo.GetInvitationByToken(invitationID) // Using ID as token or token itself
	if err != nil || invite == nil {
		return errors.New("invitation not found")
	}

	if invite.Status != models.InvitationPending {
		return errors.New("invitation already processed")
	}

	// Update Invitation
	invite.Status = models.InvitationAccepted
	if err := s.inviteRepo.UpdateInvitation(invite); err != nil {
		return err
	}

	// Link Property
	if invite.PropertyID != nil {
		prop, err := s.propRepo.GetPropertyByID(invite.PropertyID.String())
		if err == nil && prop != nil {
			uid, err := uuid.Parse(agentID)
			if err != nil {
				return err
			}
			prop.AgentID = &uid
			return s.propRepo.UpdateProperty(prop)
		}
	}

	return nil
}

func (s *agentService) AcceptMandate(propertyID, agentID string) error {
	prop, err := s.propRepo.GetPropertyByID(propertyID)
	if err != nil {
		return err
	}

	if prop.AgentID == nil || prop.AgentID.String() != agentID {
		return errors.New("this property is not assigned to you")
	}

	prop.AgentAssignmentStatus = "ACCEPTED"
	return s.propRepo.UpdateProperty(prop)
}

func (s *agentService) GetAgentDashboardSummary(agentID, email string) (any, error) {
	allMandates, _ := s.propRepo.GetByAgentID(agentID)
	invites, _ := s.inviteRepo.FindByEmail(email)
	
	var activeMandates []models.Property
	var pendingMandates []models.Property

	for _, m := range allMandates {
		if m.AgentAssignmentStatus == "ACCEPTED" {
			activeMandates = append(activeMandates, m)
		} else if m.AgentAssignmentStatus == "PENDING" {
			pendingMandates = append(pendingMandates, m)
		}
	}

	pendingInvites := 0
	for _, inv := range invites {
		if inv.Status == models.InvitationPending {
			pendingInvites++
		}
	}

	return map[string]interface{}{
		"total_managed":    len(activeMandates),
		"pending_mandates": pendingMandates,
		"active_mandates":  activeMandates,
		"new_invites":     pendingInvites,
	}, nil
}
