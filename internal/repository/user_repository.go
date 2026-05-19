package repository

import (
	"errors"

	"real-estate-backend/internal/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(user *models.User) error
	UpdateUser(user *models.User) error
	FindByEmail(email string) (*models.User, error)
	FindByEmailAndRole(email string, role models.Role) (*models.User, error)
	FindByID(id string) (*models.User, error)
	FindByVerificationToken(token string) (*models.User, error)
	FindByResetToken(token string) (*models.User, error)
	FindByRefreshToken(token string) (*models.User, error)
	FindAllAgents() ([]models.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db}
}

func (r *userRepository) CreateUser(user *models.User) error {
	// Create user and associated profile in a transaction
	return r.db.Create(user).Error
}

func (r *userRepository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Profile").Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil when not found
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByEmailAndRole(email string, role models.Role) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Profile").Where("email = ? AND role = ?", email, role).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Profile").Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByVerificationToken(token string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Profile").Where("verification_token = ?", token).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByResetToken(token string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Profile").Where("reset_token = ?", token).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByRefreshToken(token string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Profile").Where("refresh_token = ?", token).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindAllAgents() ([]models.User, error) {
	var agents []models.User
	err := r.db.Preload("Profile").
		Where("role = ? AND status = ?", models.RoleAgent, models.StatusActive).
		Find(&agents).Error
	return agents, err
}
