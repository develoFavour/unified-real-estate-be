package handler

import (
	"real-estate-backend/internal/service"
	"real-estate-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type AgentHandler struct {
	service service.AgentService
}

func NewAgentHandler(service service.AgentService) *AgentHandler {
	return &AgentHandler{service}
}

func (h *AgentHandler) GetDashboardSummary(c *fiber.Ctx) error {
	agentID := c.Locals("user_id").(string)
	
	email, ok := c.Locals("email").(string)
	if !ok {
		email = "" // Handle missing email gracefully
	}
	
	summary, err := h.service.GetAgentDashboardSummary(agentID, email)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch summary")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Summary retrieved", summary)
}

func (h *AgentHandler) GetInvitations(c *fiber.Ctx) error {
	email, ok := c.Locals("email").(string)
	if !ok {
		return utils.SuccessResponse(c, fiber.StatusOK, "Invitations retrieved", []any{})
	}
	invites, err := h.service.GetAgentInvitations(email)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch invitations")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Invitations retrieved", invites)
}

func (h *AgentHandler) AcceptInvitation(c *fiber.Ctx) error {
	agentID := c.Locals("user_id").(string)
	invitationID := c.Params("id")

	if err := h.service.AcceptInvitation(invitationID, agentID); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Invitation accepted successfully", nil)
}

func (h *AgentHandler) AcceptMandate(c *fiber.Ctx) error {
	agentID := c.Locals("user_id").(string)
	propertyID := c.Params("id")

	if err := h.service.AcceptMandate(propertyID, agentID); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Mandate accepted successfully", nil)
}
