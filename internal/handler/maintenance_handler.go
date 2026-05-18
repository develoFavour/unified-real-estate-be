package handler

import (
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/service"
	"real-estate-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type MaintenanceHandler struct {
	service service.MaintenanceService
}

func NewMaintenanceHandler(service service.MaintenanceService) *MaintenanceHandler {
	return &MaintenanceHandler{service}
}

func (h *MaintenanceHandler) GetOwnerRequests(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	requests, err := h.service.GetOwnerRequests(ownerID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch maintenance requests")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Requests retrieved", requests)
}

func (h *MaintenanceHandler) GetAgentRequests(c *fiber.Ctx) error {
	agentID := c.Locals("user_id").(string)
	requests, err := h.service.GetAgentRequests(agentID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch maintenance requests")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Requests retrieved", requests)
}

func (h *MaintenanceHandler) UpdateStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	var req struct {
		Status models.MaintenanceStatus `json:"status"`
		Note   string                   `json:"note"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid status")
	}
	role := c.Locals("role").(string)
	if role != string(models.RoleOwner) && role != string(models.RoleAgent) {
		return utils.ErrorResponse(c, fiber.StatusForbidden, "Only owners and agents can update maintenance status")
	}
	actorID := c.Locals("user_id").(string)

	if err := h.service.UpdateStatus(id, req.Status, req.Note, actorID); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Status updated successfully", nil)
}

func (h *MaintenanceHandler) CloseRequest(c *fiber.Ctx) error {
	id := c.Params("id")
	tenantID := c.Locals("user_id").(string)

	if err := h.service.CloseRequest(id, tenantID); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Maintenance request closed", nil)
}

func (h *MaintenanceHandler) ReopenRequest(c *fiber.Ctx) error {
	id := c.Params("id")
	tenantID := c.Locals("user_id").(string)

	if err := h.service.ReopenRequest(id, tenantID); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Maintenance request reopened", nil)
}

func (h *MaintenanceHandler) CreateRequest(c *fiber.Ctx) error {
	tenantID := c.Locals("user_id").(string)
	var req models.MaintenanceRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	uid, _ := uuid.Parse(tenantID)
	req.TenantID = uid

	if err := h.service.CreateRequest(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to create maintenance request")
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, "Maintenance request created successfully", req)
}
