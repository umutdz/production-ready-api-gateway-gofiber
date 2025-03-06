package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-gateway/pkg/metrics"
)

// Metrics returns a middleware that collects Prometheus metrics
func Metrics(handler *metrics.PrometheusHandler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Start timer
		start := time.Now()

		// Get request method and path
		method := c.Method()
		path := c.Path()

		// Get request size
		requestSize := float64(len(c.Request().Body()))

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get response size
		responseSize := float64(len(c.Response().Body()))

		// Get status code
		statusCode := c.Response().StatusCode()

		// Record metrics
		handler.IncRequestsTotal(method, path, statusCode)
		handler.ObserveRequestDuration(method, path, duration)
		handler.ObserveRequestSize(method, path, requestSize)
		handler.ObserveResponseSize(method, path, responseSize)

		return err
	}
}

// StatusMessage returns the HTTP status message for the given status code
func StatusMessage(status int) string {
	switch status {
	case fiber.StatusContinue:
		return "Continue"
	case fiber.StatusSwitchingProtocols:
		return "Switching Protocols"
	case fiber.StatusProcessing:
		return "Processing"
	case fiber.StatusEarlyHints:
		return "Early Hints"
	case fiber.StatusOK:
		return "OK"
	case fiber.StatusCreated:
		return "Created"
	case fiber.StatusAccepted:
		return "Accepted"
	case fiber.StatusNonAuthoritativeInformation:
		return "Non-Authoritative Information"
	case fiber.StatusNoContent:
		return "No Content"
	case fiber.StatusResetContent:
		return "Reset Content"
	case fiber.StatusPartialContent:
		return "Partial Content"
	case fiber.StatusMultiStatus:
		return "Multi-Status"
	case fiber.StatusAlreadyReported:
		return "Already Reported"
	case fiber.StatusIMUsed:
		return "IM Used"
	case fiber.StatusMultipleChoices:
		return "Multiple Choices"
	case fiber.StatusMovedPermanently:
		return "Moved Permanently"
	case fiber.StatusFound:
		return "Found"
	case fiber.StatusSeeOther:
		return "See Other"
	case fiber.StatusNotModified:
		return "Not Modified"
	case fiber.StatusUseProxy:
		return "Use Proxy"
	case fiber.StatusTemporaryRedirect:
		return "Temporary Redirect"
	case fiber.StatusPermanentRedirect:
		return "Permanent Redirect"
	case fiber.StatusBadRequest:
		return "Bad Request"
	case fiber.StatusUnauthorized:
		return "Unauthorized"
	case fiber.StatusPaymentRequired:
		return "Payment Required"
	case fiber.StatusForbidden:
		return "Forbidden"
	case fiber.StatusNotFound:
		return "Not Found"
	case fiber.StatusMethodNotAllowed:
		return "Method Not Allowed"
	case fiber.StatusNotAcceptable:
		return "Not Acceptable"
	case fiber.StatusProxyAuthRequired:
		return "Proxy Authentication Required"
	case fiber.StatusRequestTimeout:
		return "Request Timeout"
	case fiber.StatusConflict:
		return "Conflict"
	case fiber.StatusGone:
		return "Gone"
	case fiber.StatusLengthRequired:
		return "Length Required"
	case fiber.StatusPreconditionFailed:
		return "Precondition Failed"
	case fiber.StatusRequestEntityTooLarge:
		return "Request Entity Too Large"
	case fiber.StatusRequestURITooLong:
		return "Request URI Too Long"
	case fiber.StatusUnsupportedMediaType:
		return "Unsupported Media Type"
	case fiber.StatusRequestedRangeNotSatisfiable:
		return "Requested Range Not Satisfiable"
	case fiber.StatusExpectationFailed:
		return "Expectation Failed"
	case fiber.StatusTeapot:
		return "I'm a teapot"
	case fiber.StatusMisdirectedRequest:
		return "Misdirected Request"
	case fiber.StatusUnprocessableEntity:
		return "Unprocessable Entity"
	case fiber.StatusLocked:
		return "Locked"
	case fiber.StatusFailedDependency:
		return "Failed Dependency"
	case fiber.StatusTooEarly:
		return "Too Early"
	case fiber.StatusUpgradeRequired:
		return "Upgrade Required"
	case fiber.StatusPreconditionRequired:
		return "Precondition Required"
	case fiber.StatusTooManyRequests:
		return "Too Many Requests"
	case fiber.StatusRequestHeaderFieldsTooLarge:
		return "Request Header Fields Too Large"
	case fiber.StatusUnavailableForLegalReasons:
		return "Unavailable For Legal Reasons"
	case fiber.StatusInternalServerError:
		return "Internal Server Error"
	case fiber.StatusNotImplemented:
		return "Not Implemented"
	case fiber.StatusBadGateway:
		return "Bad Gateway"
	case fiber.StatusServiceUnavailable:
		return "Service Unavailable"
	case fiber.StatusGatewayTimeout:
		return "Gateway Timeout"
	case fiber.StatusHTTPVersionNotSupported:
		return "HTTP Version Not Supported"
	case fiber.StatusVariantAlsoNegotiates:
		return "Variant Also Negotiates"
	case fiber.StatusInsufficientStorage:
		return "Insufficient Storage"
	case fiber.StatusLoopDetected:
		return "Loop Detected"
	case fiber.StatusNotExtended:
		return "Not Extended"
	case fiber.StatusNetworkAuthenticationRequired:
		return "Network Authentication Required"
	default:
		return strconv.Itoa(status)
	}
}
