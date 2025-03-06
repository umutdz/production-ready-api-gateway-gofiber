package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// Security returns a middleware that adds security headers to responses
func Security() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Add security headers
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "no-referrer-when-downgrade")
		c.Set("Content-Security-Policy", "default-src 'self'")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		return c.Next()
	}
}

// CSRF returns a middleware that protects against CSRF attacks
func CSRF() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip for non-state changing methods
		if c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead || c.Method() == fiber.MethodOptions {
			return c.Next()
		}

		// Check for CSRF token
		token := c.Get("X-CSRF-Token")
		if token == "" {
			token = c.FormValue("_csrf")
		}

		// If no token is present, return an error
		if token == "" {
			return fiber.NewError(fiber.StatusForbidden, "CSRF token missing")
		}

		// TODO: Implement proper CSRF token validation
		// For now, just check if the token is not empty
		if token == "" {
			return fiber.NewError(fiber.StatusForbidden, "Invalid CSRF token")
		}

		return c.Next()
	}
}
