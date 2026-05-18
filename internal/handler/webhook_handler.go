package handler

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"real-estate-backend/internal/config"
	"real-estate-backend/internal/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WebhookHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewWebhookHandler(db *gorm.DB, cfg *config.Config) *WebhookHandler {
	return &WebhookHandler{db, cfg}
}

type PaystackEvent struct {
	Event string `json:"event"`
	Data  struct {
		Amount    float64 `json:"amount"`
		Reference string  `json:"reference"`
		Status    string  `json:"status"`
		Customer  struct {
			Email string `json:"email"`
		} `json:"customer"`
	} `json:"data"`
}

func (h *WebhookHandler) PaystackWebhook(c *fiber.Ctx) error {
	signature := c.Get("x-paystack-signature")
	body := c.Body()

	// Verify signature
	if !h.verifySignature(body, signature) {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	var event PaystackEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	if event.Event == "charge.success" {
		// 1. Find Wallet by Customer Email (via User)
		var user models.User
		if err := h.db.Preload("Wallet").Where("email = ?", event.Data.Customer.Email).First(&user).Error; err != nil {
			return c.SendStatus(fiber.StatusOK) // Return OK to Paystack even if user not found
		}

		if user.Wallet.ID != [16]byte{} {
			amount := event.Data.Amount / 100 // Convert kobo to Naira

			// 2. Start Transaction
			err := h.db.Transaction(func(tx *gorm.DB) error {
				// Create Transaction Record
				transaction := models.WalletTransaction{
					WalletID:    user.Wallet.ID,
					Amount:      amount,
					Type:        "DEPOSIT",
					Status:      "SUCCESS",
					Reference:   event.Data.Reference,
					Description: "Wallet Top-up via Paystack",
				}
				result := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "reference"}},
					DoNothing: true,
				}).Create(&transaction)
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected == 0 {
					return nil
				}

				return tx.Model(&user.Wallet).UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error
			})

			if err != nil {
				return c.SendStatus(fiber.StatusInternalServerError)
			}
		}
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *WebhookHandler) verifySignature(body []byte, signature string) bool {
	hash := hmac.New(sha512.New, []byte(h.cfg.PaystackSecretKey))
	hash.Write(body)
	expectedSignature := hex.EncodeToString(hash.Sum(nil))
	return expectedSignature == signature
}
