package handler

import (
	"real-estate-backend/internal/service"
	"real-estate-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type PaymentHandler struct {
	service service.PaymentService
}

func NewPaymentHandler(service service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service}
}

func (h *PaymentHandler) GetOwnerIncome(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	report, err := h.service.GetOwnerIncomeReport(ownerID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to generate income report")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Income report generated", report)
}
