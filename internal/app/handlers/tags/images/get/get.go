package tags_images_get

import (
	"images/internal/lib/api/response"
	"images/internal/lib/api/tag"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Response struct {
	response.Response
	ImageID string    `json:"image_id"`
	Tags    []tag.Tag `json:"tags"`
}

type TagDBGetter interface {
	GetTagsByImageID(imageID string) ([]tag.Tag, error)
}

func New(log *slog.Logger, tagDBGetter TagDBGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.tags.images.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		imageID := r.Header.Get("X-Image-ID")
		if imageID == "" {
			log.Error("image_id header is required")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("image_id header is required"))
			return
		}

		log.Info("fetching image tags", slog.String("image_id", imageID))

		tags, err := tagDBGetter.GetTagsByImageID(imageID)
		if err != nil {
			log.Error("failed to get image tags", slog.String("image_id", imageID))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to get image tags"))
			return
		}

		log.Info("image tags retrieved successfully",
			slog.String("image_id", imageID),
			slog.Int("tags_count", len(tags)))

		render.JSON(w, r, Response{
			Response: response.OK(),
			ImageID:  imageID,
			Tags:     tags,
		})
	}
}
