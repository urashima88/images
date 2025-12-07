package images_tags_attach

import (
	"images/internal/lib/api/response"
	"images/internal/lib/api/tag"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type Request struct {
	Tags []string `json:"tags"`
}

type Response struct {
	response.Response
	ImageID string    `json:"image_id"`
	Tags    []tag.Tag `json:"tags"`
}

type TagAttacher interface {
	CleanAndValidateTags(tags []string) []string
}

type TagDBAttacher interface {
	ValidateImageOwnership(profileID, imageID string) (bool, error)
	UpdateImageTags(imageID string, tagNames []string) ([]tag.Tag, error)
}

func New(log *slog.Logger, tagAttacher TagAttacher, tagDBAttacher TagDBAttacher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.images.tags.attach.New"

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

		imageID := chi.URLParam(r, "id")
		if imageID == "" {
			log.Error("image id is required in URL")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("image id is required in URL"))
			return
		}

		if _, err := uuid.Parse(imageID); err != nil {
			log.Error("invalid image id format", slog.String("image_id", imageID), slog.String("error", err.Error()))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid image id format"))
			return
		}

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", slog.String("error", err.Error()))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request body"))
			return
		}

		ownsImage, err := tagDBAttacher.ValidateImageOwnership(profileID, imageID)
		if err != nil {
			log.Error("failed to validate image ownership", slog.String("error", err.Error()))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to validate image ownership"))
			return
		}

		if !ownsImage {
			log.Warn("user doesn't own the image", slog.String("image_id", imageID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, response.Error("user doesn't have permission to tag this image"))
			return
		}

		if len(req.Tags) == 0 {
			log.Error("no tags provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("at least one tag is required"))
			return
		}

		log.Info("processing tags",
			slog.String("image_id", imageID),
			slog.Int("tags_count", len(req.Tags)))

		cleanedTags := tagAttacher.CleanAndValidateTags(req.Tags)

		createdTags, err := tagDBAttacher.UpdateImageTags(imageID, cleanedTags)
		if err != nil {
			log.Error("failed to update tags", slog.String("error", err.Error()))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to update tags"))
			return
		}

		log.Info("tags created successfully",
			slog.String("image_id", imageID),
			slog.Int("created_tags_count", len(createdTags)))

		render.JSON(w, r, Response{
			Response: response.OK(),
			ImageID:  imageID,
			Tags:     createdTags,
		})
	}

}
