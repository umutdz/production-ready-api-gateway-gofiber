package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
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
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.String("request_id", requestID),
		)

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Determine log level based on status code
		statusCode := c.Response().StatusCode()
		if statusCode >= 500 {
			logger.Error("Request completed with server error",
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.Int("status", statusCode),
				zap.Int64("duration_ms", duration.Milliseconds()),
				zap.String("request_id", requestID),
				zap.Error(err),
			)
		} else if statusCode >= 400 {
			logger.Warn("Request completed with client error",
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.Int("status", statusCode),
				zap.Int64("duration_ms", duration.Milliseconds()),
				zap.String("request_id", requestID),
				zap.Error(err),
			)
		} else {
			logger.Info("Request completed successfully",
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.Int("status", statusCode),
				zap.Int64("duration_ms", duration.Milliseconds()),
				zap.String("request_id", requestID),
			)
		}

		return err
	}
}
