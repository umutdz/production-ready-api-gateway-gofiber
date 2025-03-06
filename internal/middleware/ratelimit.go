package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RateLimiter represents a rate limiter
type RateLimiter struct {
	// Map of IP addresses to rate limit data
	ips map[string]*RateLimitData
	// Mutex for concurrent access to the map
	mu sync.RWMutex
	// Maximum number of requests per time window
	max int
	// Time window in seconds
	window time.Duration
	// Cleanup interval
	cleanupInterval time.Duration
	// Stop channel for cleanup goroutine
	stopCleanup chan bool
}

// RateLimitData represents rate limit data for an IP address
type RateLimitData struct {
	// Number of requests in the current time window
	Count int
	// Expiration time of the current time window
	Expiration time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(max int, windowSeconds int) *RateLimiter {
	limiter := &RateLimiter{
		ips:             make(map[string]*RateLimitData),
		max:             max,
		window:          time.Duration(windowSeconds) * time.Second,
		cleanupInterval: time.Minute,
		stopCleanup:     make(chan bool),
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// RateLimit returns a middleware that limits the number of requests per IP address
func RateLimit(max int, windowSeconds int) fiber.Handler {
	limiter := NewRateLimiter(max, windowSeconds)

	return func(c *fiber.Ctx) error {
		// Get client IP
		ip := c.IP()
		if ip == "" {
			ip = "unknown"
		}

		// Check if the IP is rate limited
		if limiter.isLimited(ip) {
			return fiber.NewError(fiber.StatusTooManyRequests, "Rate limit exceeded")
		}

		return c.Next()
	}
}

// isLimited checks if an IP address is rate limited
func (rl *RateLimiter) isLimited(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get current time
	now := time.Now()

	// Get rate limit data for the IP
	data, exists := rl.ips[ip]
	if !exists || now.After(data.Expiration) {
		// Create new rate limit data
		rl.ips[ip] = &RateLimitData{
			Count:      1,
			Expiration: now.Add(rl.window),
		}
		return false
	}

	// Increment request count
	data.Count++

	// Check if the IP is rate limited
	return data.Count > rl.max
}

// cleanup removes expired rate limit data
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.removeExpired()
		case <-rl.stopCleanup:
			return
		}
	}
}

// removeExpired removes expired rate limit data
func (rl *RateLimiter) removeExpired() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, data := range rl.ips {
		if now.After(data.Expiration) {
			delete(rl.ips, ip)
		}
	}
}

// Close stops the cleanup goroutine
func (rl *RateLimiter) Close() {
	rl.stopCleanup <- true
}
