package images_info

import (
	"fmt"
	app_config "images/internal/config/app-config"
	"images/internal/lib/api/image"
	"images/internal/lib/api/response"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type Request struct {
	ImageIDs []string `json:"image_ids"`
}

type Response struct {
	response.Response
	Images []image.Image `json:"images"`
}

type ImageInfoGetter interface {
	GetImageURL(imageID, extension string) string
}

type ImageInfoDBGetter interface {
	GetImagesByIDs(imageIDs []string) ([]image.Image, error)
}

func New(log *slog.Logger, imageInfoGetter ImageInfoGetter, imageInfoDBGetter ImageInfoDBGetter, imageMeta *app_config.ImageMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.images.info.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request body"))
			return
		}

		if len(req.ImageIDs) == 0 {
			log.Error("no image_ids provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("at least one image_id is required"))
			return
		}

		if len(req.ImageIDs) > imageMeta.InfoMaxNumberImages {
			log.Error("too many images", slog.Int("count", len(req.ImageIDs)))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error(fmt.Sprintf("maximum %d images per request", imageMeta.InfoMaxNumberImages)))
			return
		}

		var imageIDs []string
		for _, imageID := range req.ImageIDs {
			if _, err := uuid.Parse(imageID); err != nil {
				log.Warn("invalid image_id",
					slog.String("image_id", imageID),
					slog.String("error", err.Error()))
				continue
			}
			imageIDs = append(imageIDs, imageID)
		}

		if len(imageIDs) == 0 {
			log.Error("all image_ids are invalid")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("all provided image_ids are invalid"))
			return
		}

		log.Info("fetching images info", slog.Int("image_ids_count", len(imageIDs)))

		images, err := imageInfoDBGetter.GetImagesByIDs(imageIDs)
		if err != nil {
			log.Error("failed to get images info", slog.String("error", err.Error()))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to get images info"))
			return
		}

		if len(images) == 0 {
			log.Info("no images found", slog.Int("requested_count", len(imageIDs)))
			render.JSON(w, r, Response{
				Response: response.OK(),
				Images:   []image.Image{},
			})
			return
		}

		imagesResponse := make([]image.Image, len(images))
		for i, img := range images {
			img.FileURL = imageInfoGetter.GetImageURL(img.ImageID, img.Extension)
			imagesResponse[i] = img
		}

		log.Info("images retrieved successfully",
			slog.Int("requested_count", len(imageIDs)),
			slog.Int("found_count", len(imagesResponse)))

		render.JSON(w, r, Response{
			Response: response.OK(),
			Images:   imagesResponse,
		})
	}
}
