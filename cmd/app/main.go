package main

import (
	"context"
	"fmt"
	file_server_load "images/internal/app/handlers/file-server/load"
	post_images_download "images/internal/app/handlers/post-images/download"
	post_images_upload "images/internal/app/handlers/post-images/upload"
	tags_attach "images/internal/app/handlers/tags/attach"
	"images/internal/app/middleware/logger"
	app_config "images/internal/config/app-config"
	"images/internal/lib/logger/sl"
	image_service "images/internal/services/image-service"
	tag_service "images/internal/services/tag-service"
	"images/internal/storage/postgres"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	tagService := tag_service.New(cfg.TagMeta.MaxTagLength)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/post-images", func(r chi.Router) {
		r.Post("/upload", post_images_upload.New(log, imageService, storage, &cfg.ImageMeta))
		r.Get("/download", post_images_download.New(log, imageService, storage))
	})

	router.Route("/tags", func(r chi.Router) {
		r.Post("/attach", tags_attach.New(log, tagService, storage))
	})

	router.Handle("/images/*", http.StripPrefix("/images/", file_server_load.New(log, &cfg.FileServer, http.Dir(cfg.ImageMeta.ImageDirectory))))

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Host+":"+cfg.HTTPServer.Port))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Host + ":" + cfg.HTTPServer.Port,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Error("failed to start server")
		}
	}()

	log.Info("server started")

	<-done
	log.Info("stopping server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("failed to stop server", sl.Err(err))
		return
	}
	log.Info("server stopped")
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
