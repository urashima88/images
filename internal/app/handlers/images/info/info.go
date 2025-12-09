package images_info

import (
	"fmt"
	app_config "images/internal/config/app-config"
	"images/internal/lib/api/image"
	"images/internal/lib/api/response"
	"images/internal/lib/api/tag"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type Request struct {
	Images []image.ImageWithTags `json:"images"`
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
	GetTagsByIDs(tagIDs []string) ([]tag.Tag, error)
}

type UUIDService interface {
	CleanAndValidateIDs(ids []string) []string
}

func New(log *slog.Logger, imageInfoGetter ImageInfoGetter, imageInfoDBGetter ImageInfoDBGetter, imageMeta *app_config.ImageMeta, uuidService UUIDService) http.HandlerFunc {
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

		if len(req.Images) == 0 {
			log.Error("no images provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("at least one image is required"))
			return
		}

		if len(req.Images) > imageMeta.MaxNumberImages {
			log.Error("too many images", slog.Int("count", len(req.Images)))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error(fmt.Sprintf("maximum %d images per request", imageMeta.MaxNumberImages)))
			return
		}

		var allImageIDs []string
		var allTagIDs []string
		imageTagMap := make(map[string][]string)

		for _, imageWithTags := range req.Images {
			if _, err := uuid.Parse(imageWithTags.ImageID); err != nil {
				log.Warn("invalid image_id",
					slog.String("image_id", imageWithTags.ImageID),
					slog.String("error", err.Error()))
				continue
			}

			allImageIDs = append(allImageIDs, imageWithTags.ImageID)

			var cleanedTagIDs []string
			if len(imageWithTags.TagIDs) > 0 {
				cleanedTagIDs = uuidService.CleanAndValidateIDs(imageWithTags.TagIDs)
				allTagIDs = append(allTagIDs, cleanedTagIDs...)
			}
			imageTagMap[imageWithTags.ImageID] = cleanedTagIDs
		}

		if len(allImageIDs) == 0 {
			log.Error("all image_ids are invalid")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("all provided image_ids are invalid"))
			return
		}

		uniqueTagIDs := make(map[string]bool)
		var cleanedAllTagIDs []string
		for _, tagID := range allTagIDs {
			if !uniqueTagIDs[tagID] {
				uniqueTagIDs[tagID] = true
				cleanedAllTagIDs = append(cleanedAllTagIDs, tagID)
			}
		}

		log.Info("fetching images info",
			slog.Int("image_ids_count", len(allImageIDs)),
			slog.Int("unique_tag_ids_count", len(cleanedAllTagIDs)))

		images, err := imageInfoDBGetter.GetImagesByIDs(allImageIDs)
		if err != nil {
			log.Error("failed to get images info", slog.String("error", err.Error()))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to get images info"))
			return
		}

		if len(images) == 0 {
			log.Info("no images found", slog.Int("requested_count", len(allImageIDs)))
			render.JSON(w, r, Response{
				Response: response.OK(),
				Images:   []image.Image{},
			})
			return
		}

		var allTags []tag.Tag
		tagMap := make(map[string]tag.Tag)

		if len(cleanedAllTagIDs) > 0 {
			tags, err := imageInfoDBGetter.GetTagsByIDs(cleanedAllTagIDs)
			if err != nil {
				log.Error("failed to get tags", slog.String("error", err.Error()))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, response.Error("failed to get tags"))
				return
			}
			allTags = tags

			for _, t := range tags {
				tagMap[t.ID] = t
			}
		}

		imagesResponse := make([]image.Image, len(images))
		for i, img := range images {
			img.FileURL = imageInfoGetter.GetImageURL(img.ImageID, img.Extension)
			if tagIDs, exists := imageTagMap[img.ImageID]; exists && len(tagIDs) > 0 {
				var imageTags []tag.Tag
				for _, tagID := range tagIDs {
					if t, found := tagMap[tagID]; found {
						imageTags = append(imageTags, t)
					}
				}
				img.Tags = imageTags
			} else {
				img.Tags = []tag.Tag{}
			}
			imagesResponse[i] = img
		}

		log.Info("images retrieved successfully",
			slog.Int("requested_count", len(allImageIDs)),
			slog.Int("found_count", len(imagesResponse)),
			slog.Int("total_tags_count", len(allTags)))

		render.JSON(w, r, Response{
			Response: response.OK(),
			Images:   imagesResponse,
		})
	}
}
