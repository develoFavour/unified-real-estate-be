package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateRandomToken creates a cryptographically secure random hex string
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
// GenerateToken is a helper that returns a 32-byte secure random string
func GenerateToken() string {
	token, _ := GenerateRandomToken(32)
	return token
}
