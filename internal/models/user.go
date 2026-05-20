package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role defines the constants for RBAC
type Role string

const (
	RoleSuperAdmin Role = "SUPER_ADMIN"
	RoleAgent      Role = "AGENT"
	RoleOwner      Role = "OWNER"
	RoleTenant     Role = "TENANT"
)

type UserStatus string

const (
	StatusPending   UserStatus = "PENDING"
	StatusActive    UserStatus = "ACTIVE"
	StatusSuspended UserStatus = "SUSPENDED"
)

type User struct {
	ID                 uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Email              string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	PasswordHash       string         `gorm:"not null" json:"-"`
	Role               Role           `gorm:"type:varchar(20);not null;default:'TENANT';index:idx_users_role_status,priority:1" json:"role"`
	Status             UserStatus     `gorm:"type:varchar(20);not null;default:'ACTIVE';index:idx_users_role_status,priority:2" json:"status"`
	VerificationToken  string         `gorm:"type:varchar(100)" json:"-"`
	ResetToken         string         `gorm:"type:varchar(100)" json:"-"`
	TokenExpiry        *time.Time     `json:"-"`
	RefreshToken       string         `gorm:"type:varchar(100);index" json:"-"`
	RefreshTokenExpiry *time.Time     `json:"-"`
	Profile            Profile        `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"profile"`
	Wallet             Wallet         `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"wallet"`
	CreatedAt          time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

type Profile struct {
	ID                uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID            uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	FullName          string    `gorm:"type:varchar(100);not null" json:"full_name"`
	PhoneNumber       string    `gorm:"type:varchar(20)" json:"phone_number"`
	Address           string    `gorm:"type:text" json:"address"`
	MonthlyIncome     float64   `gorm:"type:numeric(12,2);default:0" json:"monthly_income"`
	SavingPercentage  float64   `gorm:"type:numeric(5,2);default:20" json:"saving_percentage"`
	BankName          string    `gorm:"type:varchar(100)" json:"bank_name"`
	BankAccountNumber string    `gorm:"type:varchar(20)" json:"bank_account_number"`
	AvatarURL         string    `gorm:"type:varchar(255)" json:"avatar_url"`
	NIN               string    `gorm:"type:varchar(20)" json:"nin"`
	LicenseNumber     string    `gorm:"type:varchar(50)" json:"license_number"`
	AgencyName        string    `gorm:"type:varchar(100)" json:"agency_name"`
	Nationality       string    `gorm:"type:varchar(50)" json:"nationality"`
	Bio               string    `gorm:"type:text" json:"bio"`
	Specialties       string    `gorm:"type:text" json:"specialties"` // Comma separated tags
	Rating            float64   `gorm:"default:0" json:"rating"`
	ReviewCount       int       `gorm:"default:0" json:"review_count"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
