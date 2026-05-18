package repository

import (
	"real-estate-backend/internal/models"

	"gorm.io/gorm"
)

type InvitationRepository interface {
	CreateInvitation(invitation *models.Invitation) error
	GetInvitationByToken(token string) (*models.Invitation, error)
	UpdateInvitation(invitation *models.Invitation) error
	FindByInviter(inviterID string) ([]models.Invitation, error)
	FindByEmail(email string) ([]models.Invitation, error)
}

type invitationRepository struct {
	db *gorm.DB
}

func NewInvitationRepository(db *gorm.DB) InvitationRepository {
	return &invitationRepository{db}
}

func (r *invitationRepository) CreateInvitation(invitation *models.Invitation) error {
	return r.db.Create(invitation).Error
}

func (r *invitationRepository) GetInvitationByToken(token string) (*models.Invitation, error) {
	var invitation models.Invitation
	err := r.db.Where("token = ?", token).First(&invitation).Error
	if err != nil {
		return nil, err
	}
	return &invitation, nil
}

func (r *invitationRepository) UpdateInvitation(invitation *models.Invitation) error {
	return r.db.Save(invitation).Error
}

func (r *invitationRepository) FindByInviter(inviterID string) ([]models.Invitation, error) {
	var invitations []models.Invitation
	err := r.db.Where("inviter_id = ?", inviterID).Find(&invitations).Error
	return invitations, err
}

func (r *invitationRepository) FindByEmail(email string) ([]models.Invitation, error) {
	var invitations []models.Invitation
	err := r.db.Where("email = ?", email).Find(&invitations).Error
	return invitations, err
}
