package download_post_images

import (
	"images/internal/lib/api/image"
	"images/internal/lib/api/response"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type Response struct {
	response.Response
	Images []image.DownloadImageResponse
}

type ImageDownloader interface {
	GetImageURL(imageID, extension string) string
}

type ImageDBDownloader interface {
	GetImagesByPostID(postID string) ([]image.DownloadImageResponse, error)
}

func New(log *slog.Logger, ImageDownloader ImageDownloader, imageDBDownloader ImageDBDownloader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.download_post_images.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		postID := r.Header.Get("X-Post-ID")
		if postID == "" {
			log.Error("post_id header is required")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("post_id header os required"))
			return
		}

		if _, err := uuid.Parse(postID); err != nil {
			log.Error("invalid post_id format", slog.String("post_id", postID))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid post_id format"))
			return
		}

		log.Info("fetching post images", slog.String("post_id", postID))

		imageInfos, err := imageDBDownloader.GetImagesByPostID(postID)
		if err != nil {
			log.Error("failed to get post images", slog.String("post_id", postID))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to get post images"))
			return
		}

		if len(imageInfos) == 0 {
			log.Info("no images found for post", slog.String("post_id", postID))
			render.JSON(w, r, Response{
				Response: response.OK(),
				Images:   []image.DownloadImageResponse{},
			})
			return
		}

		imagesResponse := make([]image.DownloadImageResponse, len(imageInfos))
		for i, info := range imageInfos {
			imagesResponse[i] = image.DownloadImageResponse{
				ImageID:   info.ImageID,
				Width:     info.Width,
				Height:    info.Height,
				Extension: info.Extension,
				Score:     info.Score,
				FileURL:   ImageDownloader.GetImageURL(info.ImageID, info.Extension),
			}
		}

		log.Info("post images retrieved successfully",
			slog.String("post_id", postID),
			slog.Int("images_count", len(imagesResponse)))

		render.JSON(w, r, Response{
			Response: response.OK(),
			Images:   imagesResponse,
		})
	}
}
