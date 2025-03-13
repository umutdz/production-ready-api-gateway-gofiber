package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// NewPrometheusMiddleware creates a new Prometheus middleware
func NewPrometheusMiddleware(httpRequestsTotal *prometheus.CounterVec, httpRequestDuration *prometheus.HistogramVec) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip metrics endpoint
		if c.Path() == "/metrics" {
			return c.Next()
		}

		start := time.Now()
		err := c.Next()

		// Only collect metrics if the route exists (not 404)
		if c.Route() != nil {
			method := c.Method()
			status := c.Response().StatusCode()
			duration := time.Since(start).Seconds()
			statusStr := fmt.Sprintf("%d", status)

			// Use the route path instead of the actual path for metrics
			// This prevents collecting metrics for non-existent paths
			routePath := c.Route().Path
			if routePath == "" {
				routePath = "/"
			}

			// Clean and normalize the route path
			path := cleanPath(routePath)

			httpRequestsTotal.WithLabelValues(path, method, statusStr).Inc()
			httpRequestDuration.WithLabelValues(path, method, statusStr).Observe(duration)
		}

		return err
	}
}

// cleanPath normalizes the path for metrics
func cleanPath(path string) string {
	// Remove any double slashes
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	// Remove trailing slash except for root path
	if path != "/" && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}
	return path
}
