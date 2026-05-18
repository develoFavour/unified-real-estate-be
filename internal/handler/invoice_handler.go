package handler

import (
	"real-estate-backend/internal/service"
	"real-estate-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type InvoiceHandler struct {
	service service.InvoiceService
}

func NewInvoiceHandler(service service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{service}
}

func (h *InvoiceHandler) GetMyInvoices(c *fiber.Ctx) error {
	tenantID := c.Locals("user_id").(string)

	invoices, err := h.service.GetTenantInvoices(tenantID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch invoices")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Invoices retrieved", invoices)
}

func (h *InvoiceHandler) GetIncomingInvoices(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	invoices, err := h.service.GetIncomingInvoices(userID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch invoices")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Invoices retrieved", invoices)
}

func (h *InvoiceHandler) CreateManualInvoice(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var input service.CreateManualInvoiceInput
	if err := c.BodyParser(&input); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid invoice request")
	}

	invoice, err := h.service.CreateManualInvoice(userID, input)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, "Invoice created", invoice)
}

func (h *InvoiceHandler) RunBillingCycle(c *fiber.Ctx) error {
	result, err := h.service.RunBillingCycle()
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to run billing cycle")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Billing cycle completed", result)
}
