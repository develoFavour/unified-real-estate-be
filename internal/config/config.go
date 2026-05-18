package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	BrevoAPIKey       string
	SenderEmail       string
	SenderName        string
	AppURL            string
	AllowedOrigins    string
	PaystackSecretKey string
}

func LoadConfig() (*Config, error) {
	// Try loading .env file, ignore if it doesn't exist
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	appURL := os.Getenv("APP_URL")
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000,http://127.0.0.1:3000"
		if appURL != "" {
			allowedOrigins += "," + appURL
		}
	}

	return &Config{
		Port:              port,
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		BrevoAPIKey:       os.Getenv("BREVO_API_KEY"),
		SenderEmail:       os.Getenv("SENDER_EMAIL"),
		SenderName:        os.Getenv("SENDER_NAME"),
		AppURL:            appURL,
		AllowedOrigins:    allowedOrigins,
		PaystackSecretKey: os.Getenv("PAYSTACK_SECRET_KEY"),
	}, nil
}
