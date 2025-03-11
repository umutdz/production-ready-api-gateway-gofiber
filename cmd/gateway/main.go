package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"api-gateway/internal/config"
	"api-gateway/internal/server"
	"api-gateway/pkg/logging"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Initialize logger
	logger, err := logging.NewLoggerWithConfig("info", "json", "stdout")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", zap.Error(err))
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Create and start the server
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	// Start the server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	logger.Info("API Gateway started", zap.Int("port", cfg.Server.Port))

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	if err := srv.Shutdown(); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited properly")
}
