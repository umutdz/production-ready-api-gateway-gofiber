package resilience

import (
	"time"

	"github.com/eapache/go-resiliency/retrier"

	"api-gateway/internal/config"
	"api-gateway/pkg/logging"
)

// Retrier handles retry functionality
type Retrier struct {
	retry  *retrier.Retrier
	config *config.Config
	logger *logging.Logger
}

// NewRetrier creates a new retrier
func NewRetrier(cfg *config.Config, logger *logging.Logger) (*Retrier, error) {
	// Create backoff strategy
	backoff := retrier.ExponentialBackoff(
		cfg.Resilience.MaxRetries,
		time.Duration(cfg.Resilience.RetryInterval)*time.Millisecond,
	)

	// Create retrier
	retry := retrier.New(backoff, nil)

	return &Retrier{
		retry:  retry,
		config: cfg,
		logger: logger,
	}, nil
}

// Execute executes a function with retries
func (r *Retrier) Execute(fn func() error) error {
	var lastErr error
	var attempt int

	err := r.retry.Run(func() error {
		attempt++
		err := fn()
		if err != nil {
			r.logger.Debug("Retry attempt failed",
				"attempt", attempt,
				"maxRetries", r.config.Resilience.MaxRetries,
				"error", err,
			)
			lastErr = err
			return err
		}
		return nil
	})

	if err != nil {
		r.logger.Error("All retry attempts failed",
			"maxRetries", r.config.Resilience.MaxRetries,
			"error", lastErr,
		)
		return lastErr
	}

	return nil
}
