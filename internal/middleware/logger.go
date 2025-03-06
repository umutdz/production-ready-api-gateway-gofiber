package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"api-gateway/pkg/logging"
)

// Logger returns a middleware that logs HTTP requests
func Logger(logger *logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Start timer
		start := time.Now()

		// Get request ID
		requestID := c.Get("X-Request-ID")

		// Log request
		logger.Debug("Request received",
			"method", c.Method(),
			"path", c.Path(),
			"ip", c.IP(),
			"request_id", requestID,
		)

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Determine log level based on status code
		statusCode := c.Response().StatusCode()
		if statusCode >= 500 {
			logger.Error("Request completed with server error",
				"method", c.Method(),
				"path", c.Path(),
				"status", statusCode,
				"duration_ms", duration.Milliseconds(),
				"request_id", requestID,
				"error", err,
			)
		} else if statusCode >= 400 {
			logger.Warn("Request completed with client error",
				"method", c.Method(),
				"path", c.Path(),
				"status", statusCode,
				"duration_ms", duration.Milliseconds(),
				"request_id", requestID,
			)
		} else {
			logger.Info("Request completed successfully",
				"method", c.Method(),
				"path", c.Path(),
				"status", statusCode,
				"duration_ms", duration.Milliseconds(),
				"request_id", requestID,
			)
		}

		return err
	}
}
