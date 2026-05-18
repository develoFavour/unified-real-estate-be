package utils

import "errors"

// Common Application Errors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized      = errors.New("unauthorized access")
	ErrRecordNotFound    = errors.New("record not found")
	ErrValidationFailed  = errors.New("validation failed")
)
