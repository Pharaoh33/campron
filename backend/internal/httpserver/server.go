package httpserver

import (
	"net/http"
	"time"

	"campron_enterprise/backend/internal/config"
	"campron_enterprise/backend/internal/handler"
	"campron_enterprise/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func New(cfg *config.Config, logger *zap.Logger) *http.Server {
	// In enterprise setups you usually configure mode by env (GIN_MODE=release)
	r := gin.New()

	// Middlewares
	r.Use(middleware.RequestID())
	r.Use(middleware.ZapLogger(logger))
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(cfg.Server.CorsAllowOrigin))

	// Static files: downloaded mp3
	r.Static("/files", cfg.Storage.DownloadDir)

	// API v1
	api := r.Group("/api/v1")
	{
		api.GET("/health", handler.Health())
		api.POST("/pronunciations/download", handler.DownloadPronunciation(cfg, logger))
	}

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return srv
}
