package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"campron_enterprise/backend/internal/config"
	"campron_enterprise/backend/internal/httpserver"
	"campron_enterprise/backend/internal/logging"
)

func main() {
	cfg := config.MustLoad()

	logger := logging.New(cfg)
	defer func() { _ = logger.Sync() }()

	// Make download_dir absolute for consistent behavior
	abs, err := filepath.Abs(cfg.Storage.DownloadDir)
	if err != nil {
		logger.Fatal("invalid download_dir", logging.Err(err))
	}
	cfg.Storage.DownloadDir = abs

	srv := httpserver.New(cfg, logger)

	go func() {
		logger.Info("server started",
			logging.Str("addr", cfg.Server.Addr),
			logging.Str("download_dir", cfg.Storage.DownloadDir),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", logging.Err(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("shutdown failed", logging.Err(err))
	}
	logger.Info("bye")
}
