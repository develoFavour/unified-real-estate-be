package handler

import (
	"real-estate-backend/internal/service"
	"real-estate-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type TenantHandler struct {
	service service.TenantService
}

func NewTenantHandler(service service.TenantService) *TenantHandler {
	return &TenantHandler{service}
}

func (h *TenantHandler) GetDashboard(c *fiber.Ctx) error {
	tenantID := c.Locals("user_id").(string)

	summary, err := h.service.GetTenantDashboard(tenantID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch tenant dashboard")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Dashboard retrieved", summary)
}

func (h *TenantHandler) GetPayments(c *fiber.Ctx) error {
	tenantID := c.Locals("user_id").(string)

	payments, err := h.service.GetTenantPayments(tenantID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch payments")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Payments retrieved", payments)
}
