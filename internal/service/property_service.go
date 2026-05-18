package service

import (
	"errors"
	"fmt"
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/repository"
	"real-estate-backend/pkg/mail"
	"real-estate-backend/pkg/utils"
	"time"

	"github.com/google/uuid"
)

type PropertyService interface {
	CreateProperty(req *models.Property) error
	GetPropertyByID(id string) (*models.Property, error)
	GetAllProperties(filters repository.PropertyFilters) ([]models.Property, error)
	GetByOwner(ownerID string) ([]models.Property, error)
	GetByAgent(agentID string) ([]models.Property, error)
	GetOwnedProperties(userID string) ([]models.PropertyOwnership, error)
	UpdatePropertyStatus(id string, status models.PropertyStatus) error
	UpdateSaleStatus(propertyID, userID string, status models.PropertyStatus) (*models.Property, error)
	AssignAgent(propertyID, agentEmail, inviterID string) error
	GetOwnerAgents(ownerID string) (any, error)
	GetOwnerDashboardSummary(ownerID string) (any, error)
	AcceptMandate(propertyID, agentID string) error
}

type propertyService struct {
	repo       repository.PropertyRepository
	userRepo   repository.UserRepository
	inviteRepo repository.InvitationRepository
	maintRepo  repository.MaintenanceRepository
	payRepo    repository.PaymentRepository
	mailRepo   mail.MailService
}

func NewPropertyService(
	repo repository.PropertyRepository,
	userRepo repository.UserRepository,
	inviteRepo repository.InvitationRepository,
	maintRepo repository.MaintenanceRepository,
	payRepo repository.PaymentRepository,
	mailRepo mail.MailService,
) PropertyService {
	return &propertyService{repo, userRepo, inviteRepo, maintRepo, payRepo, mailRepo}
}

func (s *propertyService) CreateProperty(req *models.Property) error {
	// Calculate Total Package (Standard Nigerian Real Estate Formula)
	// Price is base annual rent
	if req.AgencyFee == 0 {
		req.AgencyFee = req.Price * 0.10 // Default 10%
	}
	if req.LegalFee == 0 {
		req.LegalFee = req.Price * 0.10 // Default 10%
	}
	// Total Package = Rent + Agency + Legal + Caution
	req.TotalPackage = req.Price + req.AgencyFee + req.LegalFee + req.CautionDeposit

	return s.repo.CreateProperty(req)
}

func (s *propertyService) GetPropertyByID(id string) (*models.Property, error) {
	return s.repo.GetPropertyByID(id)
}

func (s *propertyService) GetAllProperties(filters repository.PropertyFilters) ([]models.Property, error) {
	return s.repo.GetAllProperties(filters)
}

func (s *propertyService) GetByOwner(ownerID string) ([]models.Property, error) {
	return s.repo.GetByOwnerID(ownerID)
}

func (s *propertyService) GetByAgent(agentID string) ([]models.Property, error) {
	return s.repo.GetByAgentID(agentID)
}

func (s *propertyService) GetOwnedProperties(userID string) ([]models.PropertyOwnership, error) {
	return s.repo.GetCurrentOwnershipsByUser(userID)
}

func (s *propertyService) UpdatePropertyStatus(id string, status models.PropertyStatus) error {
	prop, err := s.repo.GetPropertyByID(id)
	if err != nil {
		return err
	}
	prop.Status = status
	return s.repo.UpdateProperty(prop)
}

func (s *propertyService) UpdateSaleStatus(propertyID, userID string, status models.PropertyStatus) (*models.Property, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	prop, err := s.repo.GetPropertyByID(propertyID)
	if err != nil {
		return nil, err
	}
	if prop.ListingType != models.TypeSale {
		return nil, errors.New("sale status updates are only available for sale properties")
	}
	if !isPropertyStakeholderForSale(prop, userUUID) {
		return nil, errors.New("you can only update sale status for properties you own or manage")
	}
	if !isAllowedSaleStatus(status) {
		return nil, errors.New("unsupported sale status")
	}
	if !isAllowedSaleTransition(prop.Status, status) {
		return nil, fmt.Errorf("cannot move sale property from %s to %s", prop.Status, status)
	}
	if status == models.StatusSold {
		return nil, errors.New("record final settlement on the sale reservation before marking property sold")
	}

	prop.Status = status
	if err := s.repo.UpdateProperty(prop); err != nil {
		return nil, err
	}
	return s.repo.GetPropertyByID(propertyID)
}

func (s *propertyService) AssignAgent(propertyID, agentEmail, inviterID string) error {
	// 1. Find the property
	prop, err := s.repo.GetPropertyByID(propertyID)
	if err != nil {
		return err
	}

	// 2. Find if agent exists
	agent, err := s.userRepo.FindByEmailAndRole(agentEmail, models.RoleAgent)
	if err != nil {
		return err
	}

	if agent != nil {
		// Agent exists, assign directly but with PENDING status for the handshake
		prop.AgentID = &agent.ID
		prop.AgentAssignmentStatus = "PENDING"
		return s.repo.UpdateProperty(prop)
	}

	// 3. Agent doesn't exist, create an invitation
	inviterUUID, _ := uuid.Parse(inviterID)
	propUUID, _ := uuid.Parse(propertyID)
	token := utils.GenerateToken()

	invitation := &models.Invitation{
		InviterID:  inviterUUID,
		Email:      agentEmail,
		Role:       models.RoleAgent,
		PropertyID: &propUUID,
		Token:      token,
		Status:     models.InvitationPending,
		ExpiresAt:  time.Now().Add(72 * time.Hour), // 3 days
	}

	if err := s.inviteRepo.CreateInvitation(invitation); err != nil {
		return err
	}

	// 4. Send Invitation Email
	inviter, err := s.userRepo.FindByID(inviterID)
	inviterName := "An Owner"
	if err == nil && inviter != nil {
		inviterName = inviter.Profile.FullName
	}

	if err := s.mailRepo.SendAgentInvitationEmail(agentEmail, inviterName, token); err != nil {
		// Log error but don't fail assignment creation
		fmt.Printf("Error sending invitation email: %v\n", err)
	}

	return nil
}

func (s *propertyService) GetOwnerAgents(ownerID string) (any, error) {
	// 1. Get properties to find active agents
	props, err := s.repo.GetByOwnerID(ownerID)
	if err != nil {
		return nil, err
	}

	// 2. Extract unique agents
	agentMap := make(map[string]*models.User)
	propertyCount := make(map[string]int)

	for _, p := range props {
		if p.AgentID != nil {
			agentID := p.AgentID.String()
			propertyCount[agentID]++
			if _, exists := agentMap[agentID]; !exists {
				agent, err := s.userRepo.FindByID(agentID)
				if err == nil && agent != nil {
					agentMap[agentID] = agent
				}
			}
		}
	}

	// 3. Format Active Agents
	type AgentInfo struct {
		ID            string `json:"id"`
		FullName      string `json:"full_name"`
		Email         string `json:"email"`
		Phone         string `json:"phone"`
		PropertyCount int    `json:"property_count"`
		Status        string `json:"status"`
	}

	var activeAgents []AgentInfo
	for id, agent := range agentMap {
		activeAgents = append(activeAgents, AgentInfo{
			ID:            id,
			FullName:      agent.Profile.FullName,
			Email:         agent.Email,
			Phone:         agent.Profile.PhoneNumber,
			PropertyCount: propertyCount[id],
			Status:        "ACTIVE",
		})
	}

	// 4. Get Pending Invitations
	invites, err := s.inviteRepo.FindByInviter(ownerID)
	if err != nil {
		return nil, err
	}

	var pendingAgents []AgentInfo
	for _, inv := range invites {
		if inv.Status == models.InvitationPending {
			pendingAgents = append(pendingAgents, AgentInfo{
				ID:            inv.ID.String(),
				FullName:      "Invited Agent",
				Email:         inv.Email,
				Phone:         "-",
				PropertyCount: 1,
				Status:        "PENDING",
			})
		}
	}

	return map[string]interface{}{
		"active":  activeAgents,
		"pending": pendingAgents,
	}, nil
}

func (s *propertyService) GetOwnerDashboardSummary(ownerID string) (any, error) {
	// 1. Get All Properties
	props, err := s.repo.GetByOwnerID(ownerID)
	if err != nil {
		return nil, err
	}

	totalProperties := len(props)
	occupiedCount := 0
	for _, p := range props {
		if p.Status == models.StatusRented {
			occupiedCount++
		}
	}

	// 2. Get Payments (Total Revenue)
	payments, err := s.payRepo.GetByOwnerID(ownerID)
	var totalRevenue float64
	if err == nil {
		for _, p := range payments {
			totalRevenue += p.Amount
		}
	}

	// 3. Get Pending Maintenance
	maintRequests, err := s.maintRepo.GetByOwnerID(ownerID)
	pendingMaint := 0
	if err == nil {
		for _, r := range maintRequests {
			if r.Status == models.MaintenanceStatusPending {
				pendingMaint++
			}
		}
	}

	return map[string]interface{}{
		"total_properties":    totalProperties,
		"occupied_properties": occupiedCount,
		"total_revenue":       totalRevenue,
		"pending_maintenance": pendingMaint,
		"recent_payments":     payments, // Can slice this in frontend
		"maintenance":         maintRequests,
	}, nil
}

func (s *propertyService) AcceptMandate(propertyID, agentID string) error {
	prop, err := s.repo.GetPropertyByID(propertyID)
	if err != nil {
		return err
	}

	if prop.AgentID == nil || prop.AgentID.String() != agentID {
		return fmt.Errorf("this property is not assigned to you")
	}

	prop.AgentAssignmentStatus = "ACCEPTED"
	return s.repo.UpdateProperty(prop)
}

func isPropertyStakeholderForSale(prop *models.Property, userID uuid.UUID) bool {
	if prop.OwnerID != nil && *prop.OwnerID == userID {
		return true
	}
	if prop.AgentID != nil && *prop.AgentID == userID {
		return true
	}
	return false
}

func isAllowedSaleStatus(status models.PropertyStatus) bool {
	switch status {
	case models.StatusAvailable, models.StatusReserved, models.StatusUnderReview, models.StatusSold, models.StatusCancelled:
		return true
	default:
		return false
	}
}

func isAllowedSaleTransition(current, next models.PropertyStatus) bool {
	if current == next {
		return true
	}

	switch current {
	case models.StatusAvailable:
		return next == models.StatusReserved || next == models.StatusCancelled
	case models.StatusReserved:
		return next == models.StatusUnderReview || next == models.StatusAvailable || next == models.StatusCancelled
	case models.StatusUnderReview:
		return next == models.StatusSold || next == models.StatusReserved || next == models.StatusCancelled
	case models.StatusSold:
		return false
	case models.StatusCancelled:
		return next == models.StatusAvailable
	default:
		return false
	}
}
