package info

import (
	"fmt"
	app_config "images/internal/config/app-config"
	"images/internal/lib/api/image"
	"images/internal/lib/api/response"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Request struct {
	ImageIDs []string `json:"image_ids"`
}

type Response struct {
	response.Response
	Images []image.ImageInfoResponse `json:"images"`
}

type ImageInfoGetter interface {
	CleanImageIDs(imageIDs []string) []string
	GetImageURL(imageID, extension string) string
}

type ImageInfoDBGetter interface {
	GetImagesByIDs(imageIDs []string) ([]image.ImageInfoResponse, error)
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

		if len(req.ImageIDs) > imageMeta.MaxNumberImages {
			log.Error("too many image_ids", slog.Int("count", len(req.ImageIDs)))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error(fmt.Sprintf("maximum %d images for post", imageMeta.MaxNumberImages)))
			return
		}

		cleanedImageIDs := imageInfoGetter.CleanImageIDs(req.ImageIDs)
		if len(cleanedImageIDs) == 0 {
			log.Error("all image_ids are invalid")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("all provided image_ids are invalid"))
			return
		}

		log.Info("featching images info", slog.Int("image_ids_count", len(cleanedImageIDs)))

		imageInfos, err := imageInfoDBGetter.GetImagesByIDs(cleanedImageIDs)
		if err != nil {
			log.Error("failed to get images info")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to get images info"))
			return
		}

		if len(imageInfos) == 0 {
			log.Info("no images found", slog.Int("requested_count", len(cleanedImageIDs)))
			render.JSON(w, r, Response{
				Response: response.OK(),
				Images:   []image.ImageInfoResponse{},
			})
			return
		}

		imagesResponse := make([]image.ImageInfoResponse, len(imageInfos))
		for i, info := range imageInfos {
			imagesResponse[i] = image.ImageInfoResponse{
				ImageID:   info.ImageID,
				Width:     info.Width,
				Height:    info.Height,
				Extension: info.Extension,
				Tags:      info.Tags,
				CreatedAt: info.CreatedAt,
				FileURL:   imageInfoGetter.GetImageURL(info.ImageID, info.Extension),
			}
		}

		log.Info("images retrieved successfully",
			slog.Int("requested_count", len(cleanedImageIDs)),
			slog.Int("found_count", len(imagesResponse)))

		render.JSON(w, r, Response{
			Response: response.OK(),
			Images:   imagesResponse,
		})
	}
}
