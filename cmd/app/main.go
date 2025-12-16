// @title Image Service API
// @version 1.0
// @description Microservice for downloading, receiving information and storing images
//
// @host localhost:8090
// @BasePath /api/v1
//
// @tag.name Images
// @tag.description "Image operations: uploading and receiving information"
package main

import (
	"context"
	"fmt"
	file_server_load "images/internal/app/handlers/file-server/load"
	images_info "images/internal/app/handlers/images/info"
	images_upload "images/internal/app/handlers/images/upload"
	"images/internal/app/middleware/logger"
	app_config "images/internal/config/app-config"
	"images/internal/lib/logger/sl"
	image_service "images/internal/services/image-service"
	"images/internal/storage/postgres"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "images/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := app_config.MustLoad()
	log := setupLogger(cfg.Env)
	log = log.With(slog.String("env", cfg.Env))

	log.Info("starting images-app")
	log.Debug("logger debug mode enabled")

	storage, err := postgres.New(cfg)
	if err != nil {
		log.Error("failed to initialize storage", sl.Err(err))
		os.Exit(1)
	}

	imageService := image_service.New(fmt.Sprintf("http://%s:%s/%s/", cfg.FileServer.Host, cfg.HTTPServer.Port, cfg.ImageMeta.ImageDirectory))

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Post("/images", images_upload.New(log, imageService, storage, &cfg.ImageMeta))
			r.Get("/images/info", images_info.New(log, imageService, storage, &cfg.ImageMeta))
		})
	})

	router.Handle("/media/images/*", http.StripPrefix("/media/images/", file_server_load.New(log, &cfg.FileServer, http.Dir(cfg.ImageMeta.ImageDirectory))))

	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Host+":"+cfg.HTTPServer.Port))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Host + ":" + cfg.HTTPServer.Port,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info("server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("failed to shutdown server gracefully", sl.Err(err))
			if err := srv.Close(); err != nil {
				log.Error("failed to close server", sl.Err(err))
			}
		}
		log.Info("server stopped gracefully")
	case err := <-serverErrors:
		log.Error("server failed to start", sl.Err(err))
		os.Exit(1)
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}
