package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"api-gateway/internal/config"
	"api-gateway/internal/middleware"
	"api-gateway/internal/router"
	"api-gateway/pkg/logging"
	"api-gateway/pkg/metrics"
)

// Server represents the API Gateway server
type Server struct {
	app    *fiber.App
	config *config.Config
	logger *logging.Logger
	router *router.Router
}

// New creates a new server instance
func New(cfg *config.Config, logger *logging.Logger) (*Server, error) {
	// Create Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Proxy.IdleConnTimeout) * time.Second,
		AppName:      "API Gateway",
	})

	// Add built-in middleware
	app.Use(recover.New())
	app.Use(compress.New())

	// Add CORS middleware if enabled
	if cfg.Security.EnableCORS {
		app.Use(cors.New(cors.Config{
			AllowOrigins: strings.Join(cfg.Security.CORSAllowOrigins, ","),
			AllowMethods: "GET,POST,PUT,DELETE,OPTIONS,PATCH",
			AllowHeaders: "Origin,Content-Type,Accept,Authorization,Connection,Upgrade,Sec-WebSocket-Key,Sec-WebSocket-Version,Sec-WebSocket-Extensions,Sec-WebSocket-Protocol",
			AllowCredentials: false, // TODO: Change to true if we want to allow credentials
			ExposeHeaders: "Upgrade,Connection,Sec-WebSocket-Accept,Sec-WebSocket-Protocol",
		}))
	}

	// Add custom middleware
	app.Use(requestid.New())
	app.Use(middleware.Logger(logger))

	// Add security middleware if enabled
	if cfg.Security.EnableJWT {
		app.Use(middleware.JWT(cfg.Security.JWTSecret))
	}

	if cfg.Security.EnableAPIKey {
		app.Use(middleware.APIKey(cfg.Security.APIKeys))
	}

	// Add metrics middleware if enabled
	if cfg.Metrics.Enable {
		metricsHandler, err := metrics.NewPrometheusHandler()
		if err != nil {
			return nil, fmt.Errorf("failed to create metrics handler: %w", err)
		}
		app.Use(middleware.Metrics(metricsHandler))
		app.Get(cfg.Metrics.Path, metricsHandler.Handler())
	}

	// Create router
	r, err := router.New(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create router: %w", err)
	}

	// Create server
	server := &Server{
		app:    app,
		config: cfg,
		logger: logger,
		router: r,
	}

	// Register routes
	if err := server.registerRoutes(); err != nil {
		return nil, fmt.Errorf("failed to register routes: %w", err)
	}

	return server, nil
}

// Start starts the server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	if s.config.Security.EnableTLS {
		return s.app.ListenTLS(
			addr,
			s.config.Security.TLSCertFile,
			s.config.Security.TLSKeyFile,
		)
	}
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	timeout := time.Duration(s.config.Server.ShutdownTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.app.ShutdownWithContext(ctx)
}

// registerRoutes registers all routes with the router
func (s *Server) registerRoutes() error {
	// Register health check endpoint
	s.app.Get("/health", s.handleHealthCheck)

	// Register service routes
	for _, svc := range s.config.Services {
		if err := s.router.RegisterService(s.app, svc); err != nil {
			return fmt.Errorf("failed to register service %s: %w", svc.Name, err)
		}
		s.logger.Info("Registered service", "service", svc.Name)
	}

	return nil
}

// handleHealthCheck handles health check requests
func (s *Server) handleHealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}
