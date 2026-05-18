package service

import (
	"errors"
	"log"
	"time"

	"real-estate-backend/internal/models"
	"real-estate-backend/internal/repository"
	"real-estate-backend/pkg/mail"
	"real-estate-backend/pkg/token"
	"real-estate-backend/pkg/utils"
)

type AuthService interface {
	Register(req models.Profile, email, password string, role models.Role, invitationToken string) (*models.User, error)
	Login(email, password, jwtSecret string) (string, *models.User, error)
	VerifyEmail(token string) error
	ForgotPassword(email string) error
	ResetPassword(token, newPassword string) error
	GetUserByID(id string) (*models.User, error)
	GetVerifiedAgents() ([]models.User, error)
	GetInvitationByToken(token string) (*models.Invitation, error)
	UpdateProfile(userID string, profile models.Profile) error
}

type authService struct {
	repo        repository.UserRepository
	inviteRepo  repository.InvitationRepository
	propRepo    repository.PropertyRepository
	mailService mail.MailService
}

func NewAuthService(
	repo repository.UserRepository,
	inviteRepo repository.InvitationRepository,
	propRepo repository.PropertyRepository,
	mail mail.MailService,
) AuthService {
	return &authService{repo, inviteRepo, propRepo, mail}
}

func (s *authService) Register(profile models.Profile, email, password string, role models.Role, invitationToken string) (*models.User, error) {
	existingUser, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("email already in use")
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	vToken, _ := utils.GenerateRandomToken(32)

	// Determine initial status
	status := models.StatusActive
	if role == models.RoleAgent && invitationToken == "" {
		status = models.StatusPending // Only uninvited agents need admin approval
	}

	user := &models.User{
		Email:             email,
		PasswordHash:      hashedPassword,
		Role:              role,
		Status:            status,
		VerificationToken: vToken,
		Profile:           profile,
	}

	if err := s.repo.CreateUser(user); err != nil {
		return nil, err
	}

	// Handle Invitation if provided
	if invitationToken != "" && role == models.RoleAgent {
		invite, err := s.inviteRepo.GetInvitationByToken(invitationToken)
		if err == nil && invite != nil && invite.Status == models.InvitationPending {
			// Update Invitation Status
			invite.Status = models.InvitationAccepted
			s.inviteRepo.UpdateInvitation(invite)

			// Link Property if exists
			if invite.PropertyID != nil {
				prop, err := s.propRepo.GetPropertyByID(invite.PropertyID.String())
				if err == nil && prop != nil {
					prop.AgentID = &user.ID
					s.propRepo.UpdateProperty(prop)
				}
			}
		}
	}

	// Send Welcome/Verification Email
	log.Printf("Triggering Welcome Email for %s...", user.Email)
	go s.mailService.SendWelcomeEmail(user.Email, user.Profile.FullName, vToken)

	return user, nil
}

func (s *authService) Login(email, password, jwtSecret string) (string, *models.User, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return "", nil, err
	}
	if user == nil {
		return "", nil, utils.ErrInvalidCredentials
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", nil, utils.ErrInvalidCredentials
	}

	if user.Status == models.StatusPending {
		return "", nil, errors.New("account is pending approval")
	}

	jwtToken, err := token.GenerateJWT(user.ID, user.Email, string(user.Role), jwtSecret)
	if err != nil {
		return "", nil, err
	}

	return jwtToken, user, nil
}

func (s *authService) VerifyEmail(vToken string) error {
	user, err := s.repo.FindByVerificationToken(vToken)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("invalid or expired verification token")
	}

	user.VerificationToken = ""
	// Status remains pending for agents until admin approves, but we can track verification separately if needed.
	// For now, let's assume verification is part of the process.

	return s.repo.UpdateUser(user)
}

func (s *authService) ForgotPassword(email string) error {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return err
	}
	if user == nil {
		return nil // Don't leak user existence
	}

	resetToken, _ := utils.GenerateRandomToken(32)
	expiry := time.Now().Add(1 * time.Hour)

	user.ResetToken = resetToken
	user.TokenExpiry = &expiry

	if err := s.repo.UpdateUser(user); err != nil {
		return err
	}

	log.Printf("Triggering Password Reset Email for %s...", user.Email)
	go s.mailService.SendPasswordResetEmail(user.Email, user.Profile.FullName, resetToken)
	return nil
}

func (s *authService) ResetPassword(rToken, newPassword string) error {
	user, err := s.repo.FindByResetToken(rToken)
	if err != nil {
		return err
	}
	if user == nil || user.TokenExpiry == nil || time.Now().After(*user.TokenExpiry) {
		return errors.New("invalid or expired reset token")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = hashedPassword
	user.ResetToken = ""
	user.TokenExpiry = nil

	return s.repo.UpdateUser(user)
}

func (s *authService) GetUserByID(id string) (*models.User, error) {
	return s.repo.FindByID(id)
}

func (s *authService) GetVerifiedAgents() ([]models.User, error) {
	return s.repo.FindAllAgents()
}

func (s *authService) GetInvitationByToken(token string) (*models.Invitation, error) {
	invite, err := s.inviteRepo.GetInvitationByToken(token)
	if err != nil {
		return nil, err
	}
	if invite != nil && invite.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("invitation has expired")
	}
	return invite, nil
}
func (s *authService) UpdateProfile(userID string, profile models.Profile) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Update only allowed fields
	user.Profile.FullName = profile.FullName
	user.Profile.PhoneNumber = profile.PhoneNumber
	user.Profile.Address = profile.Address
	user.Profile.MonthlyIncome = profile.MonthlyIncome
	user.Profile.SavingPercentage = profile.SavingPercentage
	user.Profile.BankName = profile.BankName
	user.Profile.BankAccountNumber = profile.BankAccountNumber

	return s.repo.UpdateUser(user)
}
