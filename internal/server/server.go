package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-gateway/internal/config"
	"api-gateway/internal/middleware"
	"api-gateway/internal/router"
	"api-gateway/pkg/logging"
	"api-gateway/pkg/metrics"

	"api-gateway/pkg/tracing"

	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"go.opentelemetry.io/otel/propagation"
)

// Server represents the API Gateway server
type Server struct {
	app           *fiber.App
	config        *config.Config
	logger        *logging.Logger
	router        *router.Router
	tracerCleanup func(context.Context) error
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

	// Initialize tracer
	var tracerCleanup func(context.Context) error
	if cfg.Tracing.Enable {
		serviceName := cfg.Tracing.ServiceName
		jaegerEndpoint := cfg.Tracing.JaegerEndpoint

		logger.Info("Configuring OpenTelemetry tracing",
			zap.String("service", serviceName),
			zap.String("jaeger_endpoint", jaegerEndpoint))

		tp, cleanup := tracing.InitTracer(context.Background(), serviceName, jaegerEndpoint)

		// Use the tracer with the Fiber middleware with detailed configuration
		app.Use(otelfiber.Middleware(
			otelfiber.WithTracerProvider(tp),
			otelfiber.WithPropagators(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			)),
			otelfiber.WithServerName(serviceName),
		))

		tracerCleanup = cleanup

		logger.Info("OpenTelemetry tracing configured successfully")
	}

	// Add built-in middleware
	app.Use(recover.New())
	app.Use(compress.New())

	// Add request ID middleware first for correlation
	app.Use(requestid.New(requestid.Config{
		Header: "X-Request-ID",
		Generator: func() string {
			return uuid.New().String()
		},
	}))

	// Add custom middleware
	app.Use(middleware.Logger(logger))

	// Add CORS middleware if enabled
	if cfg.Security.EnableCORS {
		app.Use(cors.New(cors.Config{
			AllowOrigins: strings.Join(cfg.Security.CORSAllowOrigins, ","),
			AllowMethods: "GET,POST,PUT,DELETE,OPTIONS,PATCH",
			AllowHeaders: "Origin,Content-Type,Accept,Authorization,Connection,Upgrade,Sec-WebSocket-Key,Sec-WebSocket-Version,Sec-WebSocket-Extensions,Sec-WebSocket-Protocol,X-Request-ID",
			AllowCredentials: false, // TODO: Change to true if we want to allow credentials
			ExposeHeaders: "Upgrade,Connection,Sec-WebSocket-Accept,Sec-WebSocket-Protocol,X-Request-ID",
		}))
	}

	// Add security middleware if enabled
	if cfg.Security.EnableJWT {
		app.Use(middleware.JWT(cfg.Security.JWTSecret))
	}

	if cfg.Security.EnableAPIKey {
		app.Use(middleware.APIKey(cfg.Security.APIKeys))
	}

	if cfg.Metrics.Enable {
		// Create Prometheus registry
		promRegistry := prometheus.NewRegistry()
		promRegistry.MustRegister(collectors.NewGoCollector())
		promRegistry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

		// Initialize metrics collectors
		httpRequestsTotal := metrics.NewHttpRequestsTotal()
		httpRequestDuration := metrics.NewHttpRequestDuration()

		// Register metrics collectors
		promRegistry.MustRegister(httpRequestsTotal)
		promRegistry.MustRegister(httpRequestDuration)

		// HTTP requests monitoring middleware
		app.Use(middleware.NewPrometheusMiddleware(httpRequestsTotal, httpRequestDuration))

		// Metrics endpoint handler with custom registry
		app.Get("/metrics", adaptor.HTTPHandler(promhttp.HandlerFor(
			promRegistry,
			promhttp.HandlerOpts{
				Registry:          promRegistry,
				EnableOpenMetrics: true,
			},
		)))
	}

	// Create router
	r, err := router.New(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create router: %w", err)
	}

	// Create server
	server := &Server{
		app:           app,
		config:        cfg,
		logger:        logger,
		router:        r,
		tracerCleanup: tracerCleanup,
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

	// Close tracer first
	if s.tracerCleanup != nil {
		if err := s.tracerCleanup(ctx); err != nil {
			s.logger.Error("Failed to shutdown tracer", zap.Error(err))
		}
	}

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
		s.logger.Info("Registered service", zap.String("service", svc.Name))
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
