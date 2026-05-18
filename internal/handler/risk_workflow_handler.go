package handler

import (
	"real-estate-backend/internal/service"
	"real-estate-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type RiskWorkflowHandler struct {
	service service.RiskWorkflowService
}

func NewRiskWorkflowHandler(service service.RiskWorkflowService) *RiskWorkflowHandler {
	return &RiskWorkflowHandler{service: service}
}

func (h *RiskWorkflowHandler) GetMySaleReservations(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	reservations, err := h.service.GetMySaleReservations(userID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}
	return utils.SuccessResponse(c, fiber.StatusOK, "Sale reservations retrieved", reservations)
}

func (h *RiskWorkflowHandler) GetIncomingSaleReservations(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	reservations, err := h.service.GetIncomingSaleReservations(userID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}
	return utils.SuccessResponse(c, fiber.StatusOK, "Incoming sale reservations retrieved", reservations)
}

func (h *RiskWorkflowHandler) UploadDocument(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var input service.UploadDocumentInput
	if err := c.BodyParser(&input); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid document request")
	}
	document, err := h.service.UploadDocument(userID, input)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}
	return utils.SuccessResponse(c, fiber.StatusCreated, "Document uploaded", document)
}

func (h *RiskWorkflowHandler) GetDocuments(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	documents, err := h.service.GetDocuments(userID, service.GetDocumentsInput{
		LeaseID:           c.Query("lease_id"),
		InvoiceID:         c.Query("invoice_id"),
		SaleReservationID: c.Query("sale_reservation_id"),
	})
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}
	return utils.SuccessResponse(c, fiber.StatusOK, "Documents retrieved", documents)
}

func (h *RiskWorkflowHandler) AcceptLeaseDocuments(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	lease, err := h.service.AcceptLeaseDocuments(userID, c.Params("id"))
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}
	return utils.SuccessResponse(c, fiber.StatusOK, "Lease documents acknowledged", lease)
}

func (h *RiskWorkflowHandler) AcceptSaleDocuments(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	reservation, err := h.service.AcceptSaleDocuments(userID, c.Params("id"))
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}
	return utils.SuccessResponse(c, fiber.StatusOK, "Sale documents accepted", reservation)
}

func (h *RiskWorkflowHandler) RecordFinalSettlement(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var input service.FinalSettlementInput
	if err := c.BodyParser(&input); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid settlement request")
	}
	reservation, err := h.service.RecordFinalSettlement(userID, c.Params("id"), input)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}
	return utils.SuccessResponse(c, fiber.StatusOK, "Final settlement recorded", reservation)
}

func (h *RiskWorkflowHandler) OpenDispute(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var input service.OpenDisputeInput
	if err := c.BodyParser(&input); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid dispute request")
	}
	dispute, err := h.service.OpenDispute(userID, input)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}
	return utils.SuccessResponse(c, fiber.StatusCreated, "Dispute opened", dispute)
}

func (h *RiskWorkflowHandler) GetMyDisputes(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	disputes, err := h.service.GetMyDisputes(userID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}
	return utils.SuccessResponse(c, fiber.StatusOK, "Disputes retrieved", disputes)
}

func (h *RiskWorkflowHandler) RespondToDispute(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var input struct {
		Response string `json:"response"`
	}
	if err := c.BodyParser(&input); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid dispute response")
	}
	dispute, err := h.service.RespondToDispute(userID, c.Params("id"), input.Response)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}
	return utils.SuccessResponse(c, fiber.StatusOK, "Dispute response saved", dispute)
}

func (h *RiskWorkflowHandler) ResolveDispute(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("role").(string)
	var input service.ResolveDisputeInput
	if err := c.BodyParser(&input); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid dispute resolution")
	}
	dispute, err := h.service.ResolveDispute(userID, role, c.Params("id"), input)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}
	return utils.SuccessResponse(c, fiber.StatusOK, "Dispute resolved", dispute)
}

func (h *RiskWorkflowHandler) RunReservationExpiryCheck(c *fiber.Ctx) error {
	count, err := h.service.RunReservationExpiryCheck()
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to run reservation expiry check")
	}
	return utils.SuccessResponse(c, fiber.StatusOK, "Reservation expiry check completed", fiber.Map{"expired": count})
}
