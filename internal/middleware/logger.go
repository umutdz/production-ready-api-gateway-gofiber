package middleware

import (
	"time"

	"api-gateway/pkg/logging"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Logger returns a middleware that logs HTTP requests
func Logger(logger *logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Start timer
		start := time.Now()

		// Get request ID
		requestID := c.Get("X-Request-ID")

		// Get trace ID from context if available
		traceID := "unknown"
		spanContext := trace.SpanContextFromContext(c.UserContext())
		if spanContext.IsValid() {
			traceID = spanContext.TraceID().String()
		}

		// Log request
		logger.Debug("Request received",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.String("request_id", requestID),
			zap.String("trace_id", traceID),
		)

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Determine log level based on status code
		statusCode := c.Response().StatusCode()
		if c.Path() == "/health" || c.Path() == "/metrics" {
			return nil
		}
		if statusCode >= 500 {
			logger.Error("Request completed with server error",
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.Int("status", statusCode),
				zap.Int64("duration_ms", duration.Milliseconds()),
				zap.String("request_id", requestID),
				zap.String("trace_id", traceID),
				zap.Error(err),
			)
		} else if statusCode >= 400 {
			logger.Warn("Request completed with client error",
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.Int("status", statusCode),
				zap.Int64("duration_ms", duration.Milliseconds()),
				zap.String("request_id", requestID),
				zap.String("trace_id", traceID),
				zap.Error(err),
			)
		} else {
			logger.Info("Request completed successfully",
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.Int("status", statusCode),
				zap.Int64("duration_ms", duration.Milliseconds()),
				zap.String("request_id", requestID),
				zap.String("trace_id", traceID),
			)
		}

		return err
	}
}
