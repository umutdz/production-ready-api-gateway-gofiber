package resilience

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"api-gateway/internal/config"
	"api-gateway/pkg/logging"
)

// ErrTimeout is returned when a function times out
var ErrTimeout = errors.New("operation timed out")

// Timeout handles timeout functionality
type Timeout struct {
	config *config.Config
	logger *logging.Logger
}

// NewTimeout creates a new timeout handler
func NewTimeout(cfg *config.Config, logger *logging.Logger) (*Timeout, error) {
	return &Timeout{
		config: cfg,
		logger: logger,
	}, nil
}

// Execute executes a function with a timeout
func (t *Timeout) Execute(fn func() error, timeoutDuration time.Duration) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	// Create channels for result and error
	done := make(chan error, 1)

	// Execute the function in a goroutine
	go func() {
		done <- fn()
	}()

	// Wait for the function to complete or timeout
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		t.logger.Warn("Operation timed out", zap.Duration("timeout", timeoutDuration))
		return fiber.NewError(fiber.StatusGatewayTimeout, "Request timed out")
	}
}

// ExecuteWithDefaultTimeout executes a function with the default timeout
func (t *Timeout) ExecuteWithDefaultTimeout(fn func() error) error {
	return t.Execute(fn, time.Duration(t.config.Proxy.Timeout)*time.Second)
}
