package handler

import (
	"real-estate-backend/internal/config"
	"real-estate-backend/internal/models"
	"real-estate-backend/internal/service"
	"real-estate-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authService service.AuthService
	cfg         *config.Config
}

func NewAuthHandler(authService service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{authService, cfg}
}

type RegisterRequest struct {
	Email           string      `json:"email"`
	Password        string      `json:"password"`
	FullName        string      `json:"full_name"`
	Phone           string      `json:"phone"`
	Role            models.Role `json:"role"`
	Address         string      `json:"address"`
	NIN             string      `json:"nin"`
	LicenseNumber   string      `json:"license_number"`
	AgencyName      string      `json:"agency_name"`
	Nationality     string      `json:"nationality"`
	Bio             string      `json:"bio"`
	InvitationToken string      `json:"invitation_token"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request payload")
	}

	if req.Email == "" || req.Password == "" || req.FullName == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Email, password, and full name are required")
	}

	profile := models.Profile{
		FullName:      req.FullName,
		PhoneNumber:   req.Phone,
		Address:       req.Address,
		NIN:           req.NIN,
		LicenseNumber: req.LicenseNumber,
		AgencyName:    req.AgencyName,
		Nationality:   req.Nationality,
		Bio:           req.Bio,
	}

	user, err := h.authService.Register(profile, req.Email, req.Password, req.Role, req.InvitationToken)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, "Registration successful. Please check your email to verify your account.", user)
}

func (h *AuthHandler) VerifyEmail(c *fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request payload")
	}

	if req.Token == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Token is required")
	}

	if err := h.authService.VerifyEmail(req.Token); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Email verified successfully", nil)
}

func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "User not found")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Profile retrieved", user)
}

func (h *AuthHandler) ValidateInvitation(c *fiber.Ctx) error {
	token := c.Params("token")
	// We'll need a service method for this
	invitation, err := h.authService.GetInvitationByToken(token)
	if err != nil || invitation == nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid or expired invitation")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Invitation valid", invitation)
}

func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request")
	}
	if req.Email == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Email is required")
	}

	if err := h.authService.ForgotPassword(req.Email); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "If an account exists with that email, a reset link has been sent.", nil)
}

func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request")
	}
	if req.Token == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Reset token is required")
	}
	if len(req.NewPassword) < 6 {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Password must be at least 6 characters")
	}

	if err := h.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Password reset successfully", nil)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request payload")
	}

	token, user, err := h.authService.Login(req.Email, req.Password, h.cfg.JWTSecret)
	if err != nil {
		msg := "Invalid email or password"
		if err.Error() == "account is pending approval" {
			msg = err.Error()
		}
		return utils.ErrorResponse(c, fiber.StatusUnauthorized, msg)
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Login successful", fiber.Map{
		"token": token,
		"user":  user,
	})
}

func (h *AuthHandler) GetVerifiedAgents(c *fiber.Ctx) error {
	agents, err := h.authService.GetVerifiedAgents()
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch agents")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Verified agents retrieved", agents)
}
func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var profile models.Profile
	if err := c.BodyParser(&profile); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request payload")
	}

	if err := h.authService.UpdateProfile(userID, profile); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, fiber.StatusOK, "Profile updated successfully", nil)
}
