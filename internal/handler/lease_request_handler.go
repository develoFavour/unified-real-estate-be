package handler

import (
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/service"
	"real-estate-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type LeaseRequestHandler struct {
	service service.LeaseRequestService
}

func NewLeaseRequestHandler(service service.LeaseRequestService) *LeaseRequestHandler {
	return &LeaseRequestHandler{service}
}

func (h *LeaseRequestHandler) CreateRequest(c *fiber.Ctx) error {
	tenantID := c.Locals("user_id").(string)
	propertyID := c.Params("propertyID")

	var req struct {
		Message string `json:"message"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request")
	}

	leaseReq, err := h.service.CreateRequest(tenantID, propertyID, req.Message)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, "Lease request submitted", leaseReq)
}

func (h *LeaseRequestHandler) GetMyRequests(c *fiber.Ctx) error {
	tenantID := c.Locals("user_id").(string)

	requests, err := h.service.GetMyRequests(tenantID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch lease requests")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Lease requests retrieved", requests)
}

func (h *LeaseRequestHandler) GetIncomingRequests(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	requests, err := h.service.GetIncomingRequests(userID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch incoming lease requests")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Incoming lease requests retrieved", requests)
}

func (h *LeaseRequestHandler) ReviewRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	requestID := c.Params("id")

	var req struct {
		Status models.LeaseRequestStatus `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request")
	}

	leaseReq, err := h.service.ReviewRequest(userID, requestID, req.Status)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Lease request reviewed", leaseReq)
}
