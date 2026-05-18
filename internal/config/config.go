package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	BrevoAPIKey string
	SenderEmail string
	SenderName  string
	AppURL      string
	PaystackSecretKey string
}

func LoadConfig() (*Config, error) {
	// Try loading .env file, ignore if it doesn't exist
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	return &Config{
		Port:        port,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		BrevoAPIKey: os.Getenv("BREVO_API_KEY"),
		SenderEmail: os.Getenv("SENDER_EMAIL"),
		SenderName:  os.Getenv("SENDER_NAME"),
		AppURL:      os.Getenv("APP_URL"),
		PaystackSecretKey: os.Getenv("PAYSTACK_SECRET_KEY"),
	}, nil
}
