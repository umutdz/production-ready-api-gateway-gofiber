package router

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"
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
		wsPath := basePath
		wsPath = strings.TrimSuffix(wsPath, "/")

		// Dynamic WebSocket path
		websocketPath := wsPath + "/*"

		app.Get(websocketPath, func(c *fiber.Ctx) error {
			if websocket.IsWebSocketUpgrade(c) {
				// Prepare headers
				headers := make(map[string]string)
				for key, values := range c.GetReqHeaders() {
					for _, value := range values {
						if value != "" {
							if !strings.HasPrefix(strings.ToLower(key), "sec-websocket-") &&
							!strings.EqualFold(key, "Upgrade") &&
							!strings.EqualFold(key, "Connection") {
								headers[key] = value
							}
						}
					}
				}
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
				headers["X-Real-IP"] = c.IP()
				headers["X-Forwarded-For"] = c.Get("X-Forwarded-For")
				if headers["X-Forwarded-For"] == "" {
					headers["X-Forwarded-For"] = c.IP()
				}
				queryString := string(c.Context().QueryArgs().QueryString())
				if queryString != "" {
					headers["X-Original-Query"] = queryString
				}

				fullPath := c.Path()
				if queryString != "" {
					fullPath = fmt.Sprintf("%s?%s", strings.TrimSuffix(fullPath, "/*"), queryString)
				} else {
					fullPath = strings.TrimSuffix(fullPath, "/*")
				}
				c.Locals("ws_headers", headers)
				c.Locals("ws_path", fullPath)
				c.Locals("allowed", true)

				return websocket.New(func(conn *websocket.Conn) {
					wsHeaders := conn.Locals("ws_headers").(map[string]string)
					wsPath := conn.Locals("ws_path").(string)

					if err := r.handleWebSocket(conn, svc, wsPath, wsHeaders); err != nil {
						r.logger.Error("WebSocket handling error",
							zap.Error(err),
							zap.String("service", svc.Name),
							zap.String("path", wsPath))
					}
				}, websocket.Config{
					HandshakeTimeout: 10 * time.Second,
				})(c)
			}
			return c.Next()
		})
	}

	// Register HTTP routes
	// Add trailing slash if not present for HTTP path matching
	if !strings.HasSuffix(basePath, "/") {
		basePath = basePath + "/"
	}

	app.All(basePath+"*", func(c *fiber.Ctx) error {
		r.logger.Info("Handling request",
		zap.String("method", c.Method()),
		zap.String("path", c.Path()),
		zap.String("original_url", string(c.Request().URI().Path())),
		)
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

	r.logger.Info("Registered HTTP route", zap.String("service", svc.Name), zap.String("path", basePath+"*"))

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
    wsPath := path
    if svc.StripBasePath {
        wsPath = strings.TrimPrefix(path, svc.BasePath)
        if wsPath == "" {
            wsPath = "/" // Boşsa kök dizine yönlendir
        }
    }

    // İç servis için varsayılan WebSocket yolunu ekle (örneğin /socket.io)
    if !strings.HasPrefix(wsPath, "/socket.io") {
        wsPath = "/socket.io" + wsPath
    }
	// Proxy WebSocket connection
	if err := r.wsProxy.ProxyWebSocket(c, target, wsPath, headers); err != nil {
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
