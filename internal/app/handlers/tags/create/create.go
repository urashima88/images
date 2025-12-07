package tags_create

import (
	"images/internal/lib/api/response"
	"images/internal/lib/api/tag"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type Request struct {
	Tags []string `json:"tags"`
}

type Response struct {
	response.Response
	Tags []tag.Tag `json:"tags"`
}

type TagCreator interface {
	CleanAndValidateTags(tags []string) []string
}

type TagDBCreator interface {
	CreateTags(tagNames []string) ([]tag.Tag, error)
}

func New(log *slog.Logger, tagCreator TagCreator, tagDBCreator TagDBCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.tags.create.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		profileID := r.Header.Get("X-Profile-ID")
		if profileID == "" {
			log.Error("X-Profile-ID header is required")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("X-Profile-ID header is required"))
			return
		}

		if _, err := uuid.Parse(profileID); err != nil {
			log.Error("invalid profile id format", slog.String("error", err.Error()))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid profile id format"))
			return
		}

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", slog.String("error", err.Error()))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request body"))
			return
		}

		if len(req.Tags) == 0 {
			log.Error("no tags provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("at least one tag is required"))
			return
		}

		log.Info("processing tags", slog.Int("tags_count", len(req.Tags)))

		cleanedTags := tagCreator.CleanAndValidateTags(req.Tags)
		if len(cleanedTags) == 0 {
			log.Error("no valid tags provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("no valid tags provided"))
			return
		}

		createdTags, err := tagDBCreator.CreateTags(cleanedTags)
		if err != nil {
			log.Error("failed to create tags", slog.String("error", err.Error()))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to create tags"))
			return
		}

		log.Info("tags processed successfully", slog.Int("created_tags_count", len(createdTags)))

		render.JSON(w, r, Response{
			Response: response.OK(),
			Tags:     createdTags,
		})
	}
}
