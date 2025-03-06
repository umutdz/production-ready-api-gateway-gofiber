package metrics

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// PrometheusHandler handles Prometheus metrics
type PrometheusHandler struct {
	registry        *prometheus.Registry
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestSize     *prometheus.SummaryVec
	responseSize    *prometheus.SummaryVec
}

// NewPrometheusHandler creates a new Prometheus handler
func NewPrometheusHandler() (*PrometheusHandler, error) {
	registry := prometheus.NewRegistry()

	// Register the Go collector (collects runtime metrics about the Go process)
	registry.MustRegister(prometheus.NewGoCollector())

	// Create metrics
	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	requestSize := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_size_bytes",
			Help: "HTTP request size in bytes",
		},
		[]string{"method", "path"},
	)

	responseSize := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_response_size_bytes",
			Help: "HTTP response size in bytes",
		},
		[]string{"method", "path"},
	)

	// Register metrics
	registry.MustRegister(requestsTotal, requestDuration, requestSize, responseSize)

	return &PrometheusHandler{
		registry:        registry,
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
		requestSize:     requestSize,
		responseSize:    responseSize,
	}, nil
}

// Handler returns a Fiber handler for Prometheus metrics
func (p *PrometheusHandler) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create a http.Handler from promhttp
		handler := promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})

		// Convert http.Handler to fasthttp.RequestHandler
		fasthttpHandler := fasthttpadaptor.NewFastHTTPHandler(handler)

		// Call the fasthttp handler
		fasthttpHandler(c.Context())

		return nil
	}
}

// IncRequestsTotal increments the requests total counter
func (p *PrometheusHandler) IncRequestsTotal(method, path string, status int) {
	p.requestsTotal.WithLabelValues(method, path, statusMessage(status)).Inc()
}

// ObserveRequestDuration observes the request duration
func (p *PrometheusHandler) ObserveRequestDuration(method, path string, duration float64) {
	p.requestDuration.WithLabelValues(method, path).Observe(duration)
}

// ObserveRequestSize observes the request size
func (p *PrometheusHandler) ObserveRequestSize(method, path string, size float64) {
	p.requestSize.WithLabelValues(method, path).Observe(size)
}

// ObserveResponseSize observes the response size
func (p *PrometheusHandler) ObserveResponseSize(method, path string, size float64) {
	p.responseSize.WithLabelValues(method, path).Observe(size)
}

// statusMessage returns a text for the HTTP status code
func statusMessage(status int) string {
	switch status {
	case fiber.StatusOK:
		return "OK"
	case fiber.StatusCreated:
		return "Created"
	case fiber.StatusAccepted:
		return "Accepted"
	case fiber.StatusNoContent:
		return "No Content"
	case fiber.StatusBadRequest:
		return "Bad Request"
	case fiber.StatusUnauthorized:
		return "Unauthorized"
	case fiber.StatusForbidden:
		return "Forbidden"
	case fiber.StatusNotFound:
		return "Not Found"
	case fiber.StatusMethodNotAllowed:
		return "Method Not Allowed"
	case fiber.StatusInternalServerError:
		return "Internal Server Error"
	case fiber.StatusBadGateway:
		return "Bad Gateway"
	case fiber.StatusServiceUnavailable:
		return "Service Unavailable"
	case fiber.StatusGatewayTimeout:
		return "Gateway Timeout"
	default:
		return strconv.Itoa(status)
	}
}
