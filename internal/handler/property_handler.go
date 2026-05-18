package handler

import (
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/repository"
	"real-estate-backend/internal/service"
	"real-estate-backend/pkg/utils"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type PropertyHandler struct {
	propService service.PropertyService
}

func NewPropertyHandler(propService service.PropertyService) *PropertyHandler {
	return &PropertyHandler{propService}
}

type CreatePropertyRequest struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Price              float64  `json:"price"` // Annual Rent
	AgencyFee          float64  `json:"agency_fee"`
	LegalFee           float64  `json:"legal_fee"`
	CautionDeposit     float64  `json:"caution_deposit"`
	MandateType        string   `json:"mandate_type"`
	MandateDocumentURL string   `json:"mandate_document_url"`
	Address            string   `json:"address"`
	City               string   `json:"city"`
	State              string   `json:"state"`
	PropertyType       string   `json:"property_type"`
	Latitude           float64  `json:"latitude"`
	Longitude          float64  `json:"longitude"`
	ListingType        string   `json:"listing_type"`
	TotalSalePrice     float64  `json:"total_sale_price"`
	MinimumHoldingFee  float64  `json:"minimum_holding_fee"`
	IsOffPlan          bool     `json:"is_off_plan"`
	Bedrooms           *int     `json:"bedrooms"`
	Bathrooms          *int     `json:"bathrooms"`
	SquareFeet         *float64 `json:"square_feet"`
	YearBuilt          *int     `json:"year_built"`
	Amenities          string   `json:"amenities"`
	Images             []string `json:"images"`
}

func (h *PropertyHandler) CreateProperty(c *fiber.Ctx) error {
	var req CreatePropertyRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request payload")
	}

	userIDStr := c.Locals("user_id").(string)
	role := c.Locals("role").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusUnauthorized, "Invalid user token")
	}

	// Map Images
	var propertyImages []models.PropertyImage
	for _, url := range req.Images {
		propertyImages = append(propertyImages, models.PropertyImage{
			ImageURL: url,
		})
	}

	property := &models.Property{
		Title:              req.Title,
		Description:        req.Description,
		Price:              req.Price,
		ListingType:        models.ListingType(req.ListingType),
		TotalSalePrice:     req.TotalSalePrice,
		MinimumHoldingFee:  req.MinimumHoldingFee,
		IsOffPlan:          req.IsOffPlan,
		AgencyFee:          req.AgencyFee,
		LegalFee:           req.LegalFee,
		CautionDeposit:     req.CautionDeposit,
		MandateType:        req.MandateType,
		MandateDocumentURL: req.MandateDocumentURL,
		Address:            req.Address,
		City:               req.City,
		State:              req.State,
		PropertyType:       req.PropertyType,
		Latitude:           req.Latitude,
		Longitude:          req.Longitude,
		Bedrooms:           req.Bedrooms,
		Bathrooms:          req.Bathrooms,
		SquareFeet:         req.SquareFeet,
		YearBuilt:          req.YearBuilt,
		Amenities:          req.Amenities,
		Images:             propertyImages,
		Status:             models.StatusAvailable,
	}

	if role == "OWNER" {
		property.OwnerID = &userID
	} else if role == "AGENT" {
		property.AgentID = &userID
	}

	if err := h.propService.CreateProperty(property); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to create property")
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, "Property created successfully", property)
}

func (h *PropertyHandler) GetAllProperties(c *fiber.Ctx) error {
	minPrice, _ := strconv.ParseFloat(c.Query("min_price"), 64)
	maxPrice, _ := strconv.ParseFloat(c.Query("max_price"), 64)
	bedrooms, _ := strconv.Atoi(c.Query("bedrooms"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	properties, err := h.propService.GetAllProperties(repository.PropertyFilters{
		Search:       c.Query("search"),
		PropertyType: c.Query("property_type"),
		ListingType:  c.Query("listing_type"),
		Status:       c.Query("status"),
		MinPrice:     minPrice,
		MaxPrice:     maxPrice,
		Bedrooms:     bedrooms,
		Limit:        limit,
	})
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch properties")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Properties retrieved", properties)
}

func (h *PropertyHandler) GetMyProperties(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("role").(string)

	var properties []models.Property
	var err error

	if role == "OWNER" {
		properties, err = h.propService.GetByOwner(userID)
	} else if role == "AGENT" {
		properties, err = h.propService.GetByAgent(userID)
	} else {
		return utils.ErrorResponse(c, fiber.StatusForbidden, "Only owners and agents can view their properties")
	}

	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch your properties")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Your properties retrieved", properties)
}

func (h *PropertyHandler) GetOwnedProperties(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ownerships, err := h.propService.GetOwnedProperties(userID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch owned properties")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Owned properties retrieved", ownerships)
}

func (h *PropertyHandler) AssignAgent(c *fiber.Ctx) error {
	propertyID := c.Params("id")
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request payload")
	}

	ownerID := c.Locals("user_id").(string)

	if err := h.propService.AssignAgent(propertyID, req.Email, ownerID); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to assign or invite agent: "+err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Agent assignment/invitation processed", nil)
}

func (h *PropertyHandler) GetProperty(c *fiber.Ctx) error {
	id := c.Params("id")
	property, err := h.propService.GetPropertyByID(id)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "Property not found")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Property retrieved", property)
}

func (h *PropertyHandler) GetOwnerAgents(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	agents, err := h.propService.GetOwnerAgents(ownerID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch agents")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Agents retrieved", agents)
}

func (h *PropertyHandler) GetOwnerSummary(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	summary, err := h.propService.GetOwnerDashboardSummary(ownerID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch dashboard summary")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Dashboard summary retrieved", summary)
}

func (h *PropertyHandler) UpdateSaleStatus(c *fiber.Ctx) error {
	propertyID := c.Params("id")
	userID := c.Locals("user_id").(string)

	var req struct {
		Status models.PropertyStatus `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request payload")
	}

	property, err := h.propService.UpdateSaleStatus(propertyID, userID, req.Status)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Sale status updated", property)
}
