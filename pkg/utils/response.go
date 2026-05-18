package utils

import "github.com/gofiber/fiber/v2"

// SuccessResponse formats a successful API response
func SuccessResponse(c *fiber.Ctx, status int, message string, data interface{}) error {
	return c.Status(status).JSON(fiber.Map{
		"status":  "success",
		"message": message,
		"data":    data,
	})
}

// ErrorResponse formats an error API response
func ErrorResponse(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"status":  "error",
		"message": message,
	})
}
