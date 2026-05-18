package handler

import (
	"real-estate-backend/internal/service"
	"real-estate-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type WalletHandler struct {
	walletService service.WalletService
}

func NewWalletHandler(walletService service.WalletService) *WalletHandler {
	return &WalletHandler{walletService}
}

func (h *WalletHandler) GetMyWallet(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID format")
	}

	wallet, err := h.walletService.GetOrCreateWallet(userID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch wallet")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Wallet fetched", wallet)
}

func (h *WalletHandler) GetTransactions(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID format")
	}

	wallet, err := h.walletService.GetWalletByUserID(userID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "Wallet not found")
	}

	txs, err := h.walletService.GetWalletTransactions(wallet.ID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch transactions")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Transactions fetched", txs)
}

func (h *WalletHandler) GetSmartSavingTimeline(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID format")
	}
	propertyIDStr := c.Query("property_id")

	if propertyIDStr == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Property ID is required")
	}

	propertyID, err := uuid.Parse(propertyIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid Property ID")
	}

	timeline, err := h.walletService.GetSmartSavingTimeline(userID, propertyID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to calculate timeline")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Saving timeline calculated", timeline)
}
func (h *WalletHandler) Withdraw(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID")
	}

	var req struct {
		Amount float64 `json:"amount"`
		PIN    string  `json:"pin"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request")
	}

	if err := h.walletService.Withdraw(userID, req.Amount, req.PIN); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Withdrawal initiated", nil)
}

func (h *WalletHandler) PayHoldingFee(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID")
	}

	propertyIDStr := c.Params("propertyID")
	propertyID, err := uuid.Parse(propertyIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid Property ID")
	}

	var req struct {
		PIN string `json:"pin"`
	}
	_ = c.BodyParser(&req)

	if err := h.walletService.PayReservationDeposit(userID, propertyID, req.PIN); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Property reserved pending review", nil)
}

func (h *WalletHandler) ActivateVirtualAccount(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID")
	}

	wallet, err := h.walletService.ActivateVirtualAccount(userID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Virtual account generated", wallet)
}

func (h *WalletHandler) DemoFund(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID")
	}

	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request")
	}

	if err := h.walletService.DemoFund(userID, req.Amount); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Demo funds added", nil)
}

func (h *WalletHandler) InitializePayment(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID")
	}

	var req struct {
		Amount      float64 `json:"amount"`
		CallbackURL string  `json:"callback_url"`
		PIN         string  `json:"pin"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request")
	}

	payment, err := h.walletService.InitializePayment(userID, req.Amount, req.CallbackURL, req.PIN)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Payment initialized", fiber.Map{
		"authorization_url": payment.Data.AuthorizationURL,
		"access_code":       payment.Data.AccessCode,
		"reference":         payment.Data.Reference,
	})
}

func (h *WalletHandler) ConfirmPayment(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID")
	}

	var req struct {
		Reference string `json:"reference"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request")
	}

	wallet, err := h.walletService.ConfirmPayment(userID, req.Reference)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Payment confirmed", wallet)
}

func (h *WalletHandler) SetPIN(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID")
	}

	var req struct {
		PIN string `json:"pin"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request")
	}

	wallet, err := h.walletService.SetPIN(userID, req.PIN)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Wallet PIN set", wallet)
}

func (h *WalletHandler) ChangePIN(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID")
	}

	var req struct {
		CurrentPIN string `json:"current_pin"`
		NewPIN     string `json:"new_pin"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request")
	}

	wallet, err := h.walletService.ChangePIN(userID, req.CurrentPIN, req.NewPIN)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Wallet PIN changed", wallet)
}

func (h *WalletHandler) PayInvoice(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID")
	}

	invoiceIDStr := c.Params("invoiceID")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid Invoice ID")
	}

	var req struct {
		PIN string `json:"pin"`
	}
	_ = c.BodyParser(&req)

	if err := h.walletService.PayInvoice(userID, invoiceID, req.PIN); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Invoice paid successfully", nil)
}

func (h *WalletHandler) PayRent(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid User ID")
	}

	propertyIDStr := c.Params("propertyID")
	propertyID, err := uuid.Parse(propertyIDStr)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid Property ID")
	}

	var req struct {
		PIN string `json:"pin"`
	}
	_ = c.BodyParser(&req)

	if err := h.walletService.PayRent(userID, propertyID, req.PIN); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Rent paid and property secured", nil)
}
