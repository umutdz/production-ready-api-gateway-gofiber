package resilience

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sony/gobreaker"

	"api-gateway/internal/config"
	"api-gateway/pkg/logging"
)

// CircuitBreaker handles circuit breaking functionality
type CircuitBreaker struct {
	cb     *gobreaker.CircuitBreaker
	config *config.Config
	logger *logging.Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(cfg *config.Config, logger *logging.Logger) (*CircuitBreaker, error) {
	// Create circuit breaker settings
	settings := gobreaker.Settings{
		Name:        "API Gateway Circuit Breaker",
		MaxRequests: uint32(cfg.Resilience.FailureThreshold),
		Timeout:     time.Duration(cfg.Resilience.ResetTimeout) * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= uint32(cfg.Resilience.FailureThreshold) && failureRatio >= 0.5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("Circuit breaker state changed",
				"from", from.String(),
				"to", to.String(),
			)
		},
	}

	// Create circuit breaker
	cb := gobreaker.NewCircuitBreaker(settings)

	return &CircuitBreaker{
		cb:     cb,
		config: cfg,
		logger: logger,
	}, nil
}

// Execute executes a function with circuit breaking
func (c *CircuitBreaker) Execute(fn func() error) error {
	_, err := c.cb.Execute(func() (interface{}, error) {
		err := fn()
		return nil, err
	})

	if err != nil {
		// Check if the circuit is open
		if err == gobreaker.ErrOpenState {
			return fiber.NewError(fiber.StatusServiceUnavailable, "Service temporarily unavailable")
		}
		return err
	}

	return nil
}

// State returns the current state of the circuit breaker
func (c *CircuitBreaker) State() gobreaker.State {
	return c.cb.State()
}

// StateString returns the current state of the circuit breaker as a string
func (c *CircuitBreaker) StateString() string {
	state := c.cb.State()
	switch state {
	case gobreaker.StateClosed:
		return "CLOSED"
	case gobreaker.StateHalfOpen:
		return "HALF_OPEN"
	case gobreaker.StateOpen:
		return "OPEN"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", state)
	}
}
