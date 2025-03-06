package router

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	"api-gateway/internal/config"
	"api-gateway/internal/proxy"
	"api-gateway/internal/resilience"
	"api-gateway/pkg/logging"
)

// Router handles dynamic routing and service discovery
type Router struct {
	config     *config.Config
	logger     *logging.Logger
	httpProxy  *proxy.HTTPProxy
	wsProxy    *proxy.WebSocketProxy
	breaker    *resilience.CircuitBreaker
	retrier    *resilience.Retrier
}

// New creates a new router instance
func New(cfg *config.Config, logger *logging.Logger) (*Router, error) {
	// Create HTTP proxy
	httpProxy, err := proxy.NewHTTPProxy(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP proxy: %w", err)
	}

	// Create WebSocket proxy
	wsProxy, err := proxy.NewWebSocketProxy(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebSocket proxy: %w", err)
	}

	// Create circuit breaker if enabled
	var breaker *resilience.CircuitBreaker
	if cfg.Resilience.EnableCircuitBreaker {
		breaker, err = resilience.NewCircuitBreaker(cfg, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create circuit breaker: %w", err)
		}
	}

	// Create retrier if enabled
	var retrier *resilience.Retrier
	if cfg.Resilience.EnableRetry {
		retrier, err = resilience.NewRetrier(cfg, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create retrier: %w", err)
		}
	}

	return &Router{
		config:    cfg,
		logger:    logger,
		httpProxy: httpProxy,
		wsProxy:   wsProxy,
		breaker:   breaker,
		retrier:   retrier,
	}, nil
}

// RegisterService registers a service with the router
func (r *Router) RegisterService(app *fiber.App, svc config.ServiceConfig) error {
	// Create base path for the service
	basePath := svc.BasePath
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	// Special handling for WebSocket routes if enabled
	if svc.EnableWebSocket {
		// Remove trailing slash for WebSocket path matching
		wsPath := basePath
		if strings.HasSuffix(wsPath, "/") {
			wsPath = strings.TrimSuffix(wsPath, "/")
		}

		// For Socket.IO support, we need to handle the /socket.io path
		if strings.Contains(wsPath, "/consumer") {
			socketIOPath := "/socket.io/*"

			// Socket.IO handler
			app.Get(socketIOPath, func(c *fiber.Ctx) error {
				if websocket.IsWebSocketUpgrade(c) {
					r.logger.Debug("Socket.IO upgrade request detected",
						"service", svc.Name,
						"path", c.Path(),
						"original_url", c.OriginalURL(),
						"query_params", string(c.Context().QueryArgs().QueryString()),
						"headers", c.GetReqHeaders())

					// Prepare headers before upgrade
					headers := make(map[string]string)

					// Store original headers for WebSocket
					for _, key := range []string{
						"Sec-WebSocket-Key",
						"Sec-WebSocket-Version",
						"Sec-WebSocket-Extensions",
						"Sec-WebSocket-Protocol",
					} {
						if value := c.Get(key); value != "" {
							headers[key] = value
						}
					}

					// Add forwarded headers
					headers["X-Real-IP"] = c.IP()
					headers["X-Forwarded-For"] = c.Get("X-Forwarded-For")
					if headers["X-Forwarded-For"] == "" {
						headers["X-Forwarded-For"] = c.IP()
					}

					// Store query parameters
					queryString := string(c.Context().QueryArgs().QueryString())
					if queryString != "" {
						headers["X-Original-Query"] = queryString
					}

					// Get the full path including query parameters
					fullPath := c.Path()
					if queryString != "" {
						fullPath = fmt.Sprintf("%s?%s", strings.TrimSuffix(fullPath, "/*"), queryString)
					} else {
						fullPath = strings.TrimSuffix(fullPath, "/*")
					}

					r.logger.Debug("Socket.IO headers prepared",
						"headers", headers,
						"path", fullPath)

					// Store everything we need in locals
					c.Locals("ws_headers", headers)
					c.Locals("ws_path", fullPath)
					c.Locals("allowed", true)

					return websocket.New(func(conn *websocket.Conn) {
						wsHeaders := conn.Locals("ws_headers").(map[string]string)
						wsPath := conn.Locals("ws_path").(string)

						r.logger.Info("Handling Socket.IO connection",
							"service", svc.Name,
							"path", wsPath,
							"headers", wsHeaders)

						if err := r.handleWebSocket(conn, svc, wsPath, wsHeaders); err != nil {
							r.logger.Error("Socket.IO handling error",
								"error", err,
								"service", svc.Name,
								"path", wsPath)
						}
					}, websocket.Config{
						HandshakeTimeout: 10 * time.Second,
					})(c)
				}
				return c.Next()
			})
		}

		// Regular WebSocket handler
		app.Get(wsPath, func(c *fiber.Ctx) error {
			if websocket.IsWebSocketUpgrade(c) {
				r.logger.Debug("WebSocket upgrade request detected",
					"service", svc.Name,
					"path", c.Path(),
					"original_url", c.OriginalURL(),
					"base_path", basePath,
					"query_params", string(c.Context().QueryArgs().QueryString()),
					"headers", c.GetReqHeaders())

				// Prepare headers before upgrade
				headers := make(map[string]string)
				for k, v := range c.GetReqHeaders() {
					if len(v) > 0 {
						headers[k] = v[0]
					}
				}

				// Store query parameters
				queryString := string(c.Context().QueryArgs().QueryString())
				if queryString != "" {
					headers["X-Original-Query"] = queryString
				}

				// Get the path for proxying
				path := ""
				if svc.StripBasePath {
					path = strings.TrimPrefix(c.Path(), basePath)
				} else {
					path = wsPath
				}

				// Store everything we need in locals
				c.Locals("ws_headers", headers)
				c.Locals("ws_path", path)
				c.Locals("allowed", true)

				return websocket.New(func(conn *websocket.Conn) {
					wsHeaders := conn.Locals("ws_headers").(map[string]string)
					wsPath := conn.Locals("ws_path").(string)

					r.logger.Info("Handling WebSocket connection",
						"service", svc.Name,
						"path", wsPath,
						"headers", wsHeaders)

					if err := r.handleWebSocket(conn, svc, wsPath, wsHeaders); err != nil {
						r.logger.Error("WebSocket handling error",
							"error", err,
							"service", svc.Name,
							"path", wsPath)
					}
				}, websocket.Config{
					HandshakeTimeout: 10 * time.Second,
				})(c)
			}
			return c.Next()
		})

		r.logger.Info("Registered WebSocket routes",
			"service", svc.Name,
			"base_path", wsPath,
			"socket_io_enabled", strings.Contains(wsPath, "/consumer"))
	}

	// Register HTTP routes
	// Add trailing slash if not present for HTTP path matching
	if !strings.HasSuffix(basePath, "/") {
		basePath = basePath + "/"
	}

	app.All(basePath+"*", func(c *fiber.Ctx) error {
		// Skip if this is a WebSocket request that should be handled by the WebSocket handler
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}

		// Get the path without the base path if strip is enabled
		path := c.Params("*")
		if svc.StripBasePath {
			path = strings.TrimPrefix(path, basePath)
		}

		// Handle HTTP request
		return r.handleHTTP(c, svc, path)
	})

	r.logger.Info("Registered HTTP route", "service", svc.Name, "path", basePath+"*")

	return nil
}

// handleHTTP handles HTTP requests
func (r *Router) handleHTTP(c *fiber.Ctx, svc config.ServiceConfig, path string) error {
	// Add custom headers if configured
	for key, value := range svc.Headers {
		c.Request().Header.Set(key, value)
	}

	// Get target URL
	target, err := r.getTarget(svc)
	if err != nil {
		return err
	}

	// Handle request with resilience patterns if enabled
	if r.config.Resilience.EnableCircuitBreaker && r.breaker != nil {
		return r.breaker.Execute(func() error {
			if r.config.Resilience.EnableRetry && r.retrier != nil {
				return r.retrier.Execute(func() error {
					return r.httpProxy.Forward(c, target, path, svc)
				})
			}
			return r.httpProxy.Forward(c, target, path, svc)
		})
	} else if r.config.Resilience.EnableRetry && r.retrier != nil {
		return r.retrier.Execute(func() error {
			return r.httpProxy.Forward(c, target, path, svc)
		})
	}

	// Forward the request directly
	return r.httpProxy.Forward(c, target, path, svc)
}

// handleWebSocket handles WebSocket connections
func (r *Router) handleWebSocket(c *websocket.Conn, svc config.ServiceConfig, path string, headers map[string]string) error {
	// Get target service URL
	target, err := r.getTarget(svc)
	if err != nil {
		return fmt.Errorf("failed to get target service URL: %w", err)
	}

	// Proxy WebSocket connection
	if err := r.wsProxy.ProxyWebSocket(c, target, path, headers); err != nil {
		return fmt.Errorf("failed to proxy WebSocket: %w", err)
	}

	return nil
}

// getTarget returns a target URL for the service
func (r *Router) getTarget(svc config.ServiceConfig) (string, error) {
	// Simple round-robin load balancing
	if len(svc.Targets) == 0 {
		return "", fiber.NewError(fiber.StatusServiceUnavailable, "no targets available for service "+svc.Name)
	}

	// TODO: Implement more sophisticated load balancing and service discovery
	// For now, just use the first target
	return svc.Targets[0], nil
}
