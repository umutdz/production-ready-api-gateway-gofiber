package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestID returns a middleware that adds a unique request ID to each request
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if request ID already exists
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			// Generate a new request ID
			requestID = uuid.New().String()
			c.Set("X-Request-ID", requestID)
		}

		// Add request ID to response
		c.Set("X-Request-ID", requestID)

		return c.Next()
	}
}
