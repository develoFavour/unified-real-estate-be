package repository

import (
	"real-estate-backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

type PropertyFilters struct {
	Search       string
	PropertyType string
	ListingType  string
	Status       string
	MinPrice     float64
	MaxPrice     float64
	Bedrooms     int
	Limit        int
}

type PropertyRepository interface {
	CreateProperty(property *models.Property) error
	GetPropertyByID(id string) (*models.Property, error)
	GetAllProperties(filters PropertyFilters) ([]models.Property, error)
	GetByOwnerID(ownerID string) ([]models.Property, error)
	GetByAgentID(agentID string) ([]models.Property, error)
	GetCurrentOwnershipsByUser(userID string) ([]models.PropertyOwnership, error)
	UpdateProperty(property *models.Property) error
	DeleteProperty(id string) error
}

type propertyRepository struct {
	db *gorm.DB
}

func NewPropertyRepository(db *gorm.DB) PropertyRepository {
	return &propertyRepository{db}
}

func (r *propertyRepository) CreateProperty(property *models.Property) error {
	return r.db.Create(property).Error
}

func (r *propertyRepository) GetPropertyByID(id string) (*models.Property, error) {
	var property models.Property
	err := r.db.Preload("Images").
		Preload("Owner").Preload("Owner.Profile").
		Preload("Agent").Preload("Agent.Profile").
		Where("id = ?", id).First(&property).Error
	if err != nil {
		return nil, err
	}
	return &property, nil
}

func (r *propertyRepository) GetAllProperties(filters PropertyFilters) ([]models.Property, error) {
	var properties []models.Property
	query := r.db.Preload("Images").Model(&models.Property{})

	if filters.Search != "" {
		term := "%" + strings.ToLower(strings.TrimSpace(filters.Search)) + "%"
		query = query.Where(
			"LOWER(title) LIKE ? OR LOWER(address) LIKE ? OR LOWER(city) LIKE ? OR LOWER(state) LIKE ? OR LOWER(description) LIKE ?",
			term, term, term, term, term,
		)
	}

	if filters.PropertyType != "" {
		propertyType := strings.ToUpper(strings.TrimSpace(filters.PropertyType))
		categoryTerm := "%" + strings.ToLower(strings.ReplaceAll(propertyType, "_", " ")) + "%"
		if r.db.Migrator().HasColumn(&models.Property{}, "property_type") {
			query = query.Where(
				"property_type = ? OR LOWER(title) LIKE ? OR LOWER(description) LIKE ?",
				propertyType, categoryTerm, categoryTerm,
			)
		} else {
			query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ?", categoryTerm, categoryTerm)
		}
	}

	if filters.ListingType != "" {
		query = query.Where("listing_type = ?", strings.ToUpper(strings.TrimSpace(filters.ListingType)))
	}

	if filters.Status != "" {
		query = query.Where("status = ?", strings.ToUpper(strings.TrimSpace(filters.Status)))
	}

	if filters.MinPrice > 0 {
		query = query.Where("price >= ?", filters.MinPrice)
	}

	if filters.MaxPrice > 0 {
		query = query.Where("price <= ?", filters.MaxPrice)
	}

	if filters.Bedrooms > 0 {
		query = query.Where("bedrooms >= ?", filters.Bedrooms)
	}

	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 60
	}

	err := query.Order("created_at DESC").Limit(limit).Find(&properties).Error
	return properties, err
}

func (r *propertyRepository) GetByOwnerID(ownerID string) ([]models.Property, error) {
	var properties []models.Property
	err := r.db.Preload("Images").Where("owner_id = ?", ownerID).Find(&properties).Error
	return properties, err
}

func (r *propertyRepository) GetByAgentID(agentID string) ([]models.Property, error) {
	var properties []models.Property
	err := r.db.Preload("Images").
		Preload("Owner").Preload("Owner.Profile").
		Where("agent_id = ?", agentID).Find(&properties).Error
	return properties, err
}

func (r *propertyRepository) GetCurrentOwnershipsByUser(userID string) ([]models.PropertyOwnership, error) {
	var ownerships []models.PropertyOwnership
	err := r.db.Preload("Property").Preload("Property.Images").Preload("SaleReservation").
		Where("user_id = ? AND is_current = ?", userID, true).
		Order("started_at DESC").
		Find(&ownerships).Error
	return ownerships, err
}

func (r *propertyRepository) UpdateProperty(property *models.Property) error {
	return r.db.Save(property).Error
}

func (r *propertyRepository) DeleteProperty(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Property{}).Error
}
