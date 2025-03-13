package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "api_gateway"
)

var (
	defaultBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
)

// NewHttpRequestsTotal creates a new counter vector for HTTP requests
func NewHttpRequestsTotal() *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"path", "method", "status"},
	)
}

// NewHttpRequestDuration creates a new histogram vector for HTTP request durations
func NewHttpRequestDuration() *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds",
			Buckets:   defaultBuckets,
		},
		[]string{"path", "method", "status"},
	)
}

