package post_images_score_update

import (
	"images/internal/lib/api/response"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Request struct {
	Value int `json:"value"`
}

type Response struct {
	response.Response
	ImageID string `json:"image_id"`
	Score   int    `json:"score"`
}

type ScoreDBUpdater interface {
	UpdateImageScore(imageID string, value int) (int, error)
}

func New(log *slog.Logger, scoreDBUpdater ScoreDBUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.post_images.score.update.New"

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

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request body"))
			return
		}

		if req.Value != 1 && req.Value != -1 {
			log.Error("invalid value", slog.Int("value", req.Value))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("value must be 1 (upvote) or -1 (downvote)"))
			return
		}

		log.Info("updating image score",
			slog.String("image_id", imageID),
			slog.Int("value", req.Value))

		newScore, err := scoreDBUpdater.UpdateImageScore(imageID, req.Value)
		if err != nil {
			log.Error("failed to update image score", slog.String("image_id", imageID))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to update image score"))
			return
		}

		log.Info("image score updated successfully",
			slog.String("image_id", imageID),
			slog.Int("new_score", newScore))

		render.JSON(w, r, Response{
			Response: response.OK(),
			ImageID:  imageID,
			Score:    newScore,
		})
	}
}
