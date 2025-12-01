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
	ImageID string    `json:"image_id"`
	Tags    []tag.Tag `json:"tags"`
}

type TagCreator interface {
	CleanAndValidateTags(tags []string) []string
}

type TagDBCreator interface {
	ValidateImageOwnership(profileID, imageID string) (bool, error)
	CreateImageTags(imageID string, tagNames []string) ([]tag.Tag, error)
}

func New(log *slog.Logger, tagCreator TagCreator, tagDBCreator TagDBCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.tags.create.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		profileID := r.Header.Get("X-Profile-ID")
		imageID := r.Header.Get("X-Image-ID")

		if profileID == "" {
			log.Error("profile_id header is required")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("profile_id header is required"))
			return
		}

		if imageID == "" {
			log.Error("image_id header is required")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("image_id header is required"))
			return
		}

		if _, err := uuid.Parse(profileID); err != nil {
			log.Error("invalid profile_id format")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid profile_id format"))
			return
		}

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request body"))
			return
		}

		ownsImage, err := tagDBCreator.ValidateImageOwnership(profileID, imageID)
		if err != nil {
			log.Error("failed to validate image ownership")
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

		cleanedTags := tagCreator.CleanAndValidateTags(req.Tags)

		log.Info("processing tags",
			slog.String("image_id", imageID),
			slog.Int("tags_count", len(cleanedTags)))

		createdTags, err := tagDBCreator.CreateImageTags(imageID, cleanedTags)
		if err != nil {
			log.Error("failed to create tags")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to create tags"))
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
