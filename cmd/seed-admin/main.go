package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"real-estate-backend/internal/config"
	"real-estate-backend/internal/models"
	"real-estate-backend/pkg/database"
	"real-estate-backend/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func main() {
	email := flag.String("email", os.Getenv("ADMIN_EMAIL"), "admin email")
	password := flag.String("password", os.Getenv("ADMIN_PASSWORD"), "admin password")
	name := flag.String("name", os.Getenv("ADMIN_NAME"), "admin full name")
	flag.Parse()

	*email = strings.TrimSpace(*email)
	*password = strings.TrimSpace(*password)
	*name = strings.TrimSpace(*name)

	if *email == "" {
		log.Fatal("admin email is required. Use -email or ADMIN_EMAIL")
	}
	if *name == "" {
		*name = "Platform Admin"
	}

	generatedPassword := false
	if *password == "" {
		token, err := utils.GenerateRandomToken(10)
		if err != nil {
			log.Fatalf("failed to generate admin password: %v", err)
		}
		*password = token
		generatedPassword = true
	}
	if len(*password) < 6 {
		log.Fatal("admin password must be at least 6 characters")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if db == nil {
		log.Fatal("database connection is not configured")
	}

	passwordHash, err := utils.HashPassword(*password)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	var user models.User
	err = db.Preload("Profile").Where("email = ?", *email).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Fatalf("failed to check existing user: %v", err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = models.User{
			Email:        *email,
			PasswordHash: passwordHash,
			Role:         models.RoleSuperAdmin,
			Status:       models.StatusActive,
			Profile: models.Profile{
				FullName: *name,
			},
		}
		if err := db.Create(&user).Error; err != nil {
			log.Fatalf("failed to create admin user: %v", err)
		}
		fmt.Printf("Created SUPER_ADMIN user: %s\n", *email)
	} else {
		updates := map[string]any{
			"password_hash":        passwordHash,
			"role":                 models.RoleSuperAdmin,
			"status":               models.StatusActive,
			"refresh_token":        "",
			"refresh_token_expiry": nil,
			"verification_token":   "",
			"reset_token":          "",
			"token_expiry":         nil,
		}
		if err := db.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
			log.Fatalf("failed to update admin user: %v", err)
		}

		if user.Profile.ID == uuid.Nil {
			if err := db.Create(&models.Profile{UserID: user.ID, FullName: *name}).Error; err != nil {
				log.Fatalf("failed to create admin profile: %v", err)
			}
		} else if err := db.Model(&models.Profile{}).Where("user_id = ?", user.ID).Update("full_name", *name).Error; err != nil {
			log.Fatalf("failed to update admin profile: %v", err)
		}

		fmt.Printf("Updated existing user to SUPER_ADMIN: %s\n", *email)
	}

	fmt.Println("Admin login:")
	fmt.Printf("Email: %s\n", *email)
	if generatedPassword {
		fmt.Printf("Password: %s\n", *password)
	} else {
		fmt.Println("Password: [the password you provided]")
	}
}
