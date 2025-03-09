package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-gateway/internal/config"
	"api-gateway/pkg/cache"
	"api-gateway/pkg/logging"
)

// HTTPProxy handles HTTP proxying
type HTTPProxy struct {
	client    *http.Client
	config    *config.Config
	logger    *logging.Logger
	cache     *cache.Cache
}

// NewHTTPProxy creates a new HTTP proxy
func NewHTTPProxy(cfg *config.Config, logger *logging.Logger) (*HTTPProxy, error) {
	// Create HTTP client with custom transport
	transport := &http.Transport{
		MaxIdleConns:        cfg.Proxy.MaxIdleConns,
		IdleConnTimeout:     time.Duration(cfg.Proxy.IdleConnTimeout) * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.Proxy.Timeout) * time.Second,
	}

	// Create cache if enabled
	// TODO: Cache change to redis from in-memory cache
	var c *cache.Cache
	var err error
	if cfg.Proxy.EnableCache {
		c, err = cache.New(cfg.Proxy.CacheTTL)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache: %w", err)
		}
	}

	return &HTTPProxy{
		client: client,
		config: cfg,
		logger: logger,
		cache:  c,
	}, nil
}

// Forward forwards an HTTP request to the target service
func (p *HTTPProxy) Forward(c *fiber.Ctx, target, path string, svc config.ServiceConfig) error {
	// Skip WebSocket requests - they should be handled by the WebSocket proxy
	if c.Get("Upgrade") == "websocket" {
		p.logger.Debug("Skipping WebSocket request in HTTP proxy",
			"path", c.Path(),
			"service", svc.Name)
		return c.Next()
	}

	// Check if response is in cache - TODO: Cache change to redis from in-memory cache
	if p.config.Proxy.EnableCache && p.cache != nil && c.Method() == fiber.MethodGet {
		queryString := c.Request().URI().QueryString()
		cacheKey := getCacheKey(c.Path(), string(queryString))
		if cachedResp, found := p.cache.Get(cacheKey); found {
			p.logger.Debug("Cache hit", "path", c.Path(), "service", svc.Name)
			return c.Send(cachedResp.([]byte))
		}
	}

	// Parse target URL
	targetURL, err := url.Parse(target)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "invalid target URL")
	}

	// Create the request URL
	requestURL := fmt.Sprintf("%s://%s", targetURL.Scheme, targetURL.Host)
	if targetURL.Path != "" && targetURL.Path != "/" {
		requestURL = fmt.Sprintf("%s%s", requestURL, targetURL.Path)
	}
	if path != "" {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		requestURL = fmt.Sprintf("%s%s", requestURL, path)
	}
	queryString := c.Request().URI().QueryString()
	if len(queryString) > 0 {
		requestURL = fmt.Sprintf("%s?%s", requestURL, string(queryString))
	}

	// Create the request
	req, err := http.NewRequest(c.Method(), requestURL, bytes.NewReader(c.Body()))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create request")
	}

	// Copy headers
	c.Request().Header.VisitAll(func(key, value []byte) {
		req.Header.Set(string(key), string(value))
	})

	// Set host header
	req.Host = targetURL.Host

	// Add custom headers from service config
	for key, value := range svc.Headers {
		req.Header.Set(key, value)
	}

	// Execute the request
	start := time.Now()
	resp, err := p.client.Do(req)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to execute request")
	}
	defer resp.Body.Close()

	// Log the request
	p.logger.Debug("Proxied request",
		"method", c.Method(),
		"path", c.Path(),
		"target", requestURL,
		"status", resp.StatusCode,
		"duration", time.Since(start).Milliseconds(),
		"service", svc.Name,
	)

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to read response body")
	}

	// Cache the response if enabled
	if p.config.Proxy.EnableCache && p.cache != nil && c.Method() == fiber.MethodGet && resp.StatusCode == fiber.StatusOK {
		cacheKey := getCacheKey(c.Path(), string(queryString))
		p.cache.Set(cacheKey, body)
	}

	// Set response status
	c.Status(resp.StatusCode)

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Set(key, value)
		}
	}

	// Send response body
	return c.Send(body)
}

// getCacheKey generates a cache key from path and query
func getCacheKey(path, query string) string {
	if query != "" {
		return fmt.Sprintf("%s?%s", path, query)
	}
	return path
}

