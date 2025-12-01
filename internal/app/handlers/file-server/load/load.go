package file_server_load

import (
	"fmt"
	app_config "images/internal/config/app-config"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func New(log *slog.Logger, fileServer *app_config.FileServer, fileSystem http.FileSystem) http.Handler {
	fs := http.FileServer(fileSystem)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.load.New()"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%s", fileServer.CacheMaxAge))

		log.Info("serving static file", slog.String("path", r.URL.Path))

		fs.ServeHTTP(w, r)
	})
}
