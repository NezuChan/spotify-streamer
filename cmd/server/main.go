package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/nezuchan/spotify-streamer/config"
	"github.com/nezuchan/spotify-streamer/server"
	"github.com/sirupsen/logrus"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Setup logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Set log level from config
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err == nil {
		logger.SetLevel(level)
	}

	logger.Info("Starting Spotify Streamer service...")
	logger.WithField("version", "0.1.0").Info("Service information")

	// Create and start server
	srv, err := server.New(logger, cfg)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create server")
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	go func() {
		if err := srv.Start(ctx); err != nil {
			logger.WithError(err).Error("Server error")
			cancel()
		}
	}()

	logger.WithField("address", cfg.Server.Host+":"+cfg.Server.Port).Info("Server started")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.WithField("signal", sig).Info("Received shutdown signal")
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	// Shutdown gracefully
	logger.Info("Shutting down...")
	srv.Shutdown()
	logger.Info("Shutdown complete")
}
