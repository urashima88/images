package post_images_upload

import (
	"fmt"
	app_config "images/internal/config/app-config"
	"images/internal/lib/api/image"
	"images/internal/lib/api/response"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type Response struct {
	response.Response
	ImageIDs []string `json:"image_ids"`
}

type PartialSuccessResponse struct {
	response.Response
	ImageIDs     []string                    `json:"image_ids"`
	SuccessCount int                         `json:"success_count"`
	FailedCount  int                         `json:"failed_count"`
	Failed       []image.FailedImageResponse `json:"failed"`
}

type ImageUploader interface {
	GenerateImageID() string
	GetImageDimensions(imageData []byte) (int, int, error)
	SaveImageToDisk(imageData []byte, imageDir, fileName string) error
}

type ImageDBUploader interface {
	SaveImage(profileID, imageID string, width, height int, extension string) (string, error)
	SavePostImage(postID, imageID string) error
}

func New(log *slog.Logger, imageUploader ImageUploader, imageDBUploader ImageDBUploader, imageMeta *app_config.ImageMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.post_images.upload.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		profileID := r.Header.Get("X-Profile-ID")
		postID := r.Header.Get("X-Post-ID")

		if profileID == "" {
			log.Error("profile_id header is required")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("profile_id header is required"))
			return
		}

		if postID == "" {
			log.Error("post_id header is required")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("post_id header is required"))
			return
		}

		if _, err := uuid.Parse(profileID); err != nil {
			log.Error("invalid profile_id format", slog.String("profile_id", profileID))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid profile_id format"))
			return
		}

		if _, err := uuid.Parse(postID); err != nil {
			log.Error("invalid post_id format", slog.String("post_id", postID))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid post_id format"))
			return
		}

		err := r.ParseMultipartForm(imageMeta.MaxMemory << 20)
		if err != nil {
			log.Error("failed to parse multipart form", slog.String("error", err.Error()))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("failed to parse form data"))
			return
		}

		files := r.MultipartForm.File["images"]
		if len(files) == 0 {
			log.Error("no images provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("at least one image is required"))
			return
		}

		if len(files) > imageMeta.PostMaxNumberImages {
			log.Error("too many images", slog.Int("count", len(files)))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error(fmt.Sprintf("maximum %d images allowed", imageMeta.PostMaxNumberImages)))
			return
		}

		log.Info("processing images", slog.Int("image_count", len(files)))

		var uploadedImages []image.UploadImage
		var failedImages []image.FailedImageResponse

		for i, fileHeader := range files {
			img, err := processImage(fileHeader, profileID, postID, imageUploader, imageDBUploader, imageMeta)
			if err != nil {
				failedImages = append(failedImages, image.FailedImageResponse{
					FileName: fileHeader.Filename,
					Error:    err.Error(),
				})

				log.Error("failed to process image",
					slog.Int("index", i),
					slog.String("filename", fileHeader.Filename),
					slog.String("error", err.Error()))
			} else {
				uploadedImages = append(uploadedImages, img)
			}
		}

		if len(uploadedImages) == 0 {
			log.Error("all images failed to process")
			render.Status(r, http.StatusUnprocessableEntity)
			render.JSON(w, r, response.Error("failed to process any image"))
			return
		}

		if len(failedImages) > 0 {
			imageIDs := make([]string, len(uploadedImages))
			for i, img := range uploadedImages {
				imageIDs[i] = img.ImageID
			}

			log.Info("only a part of images uploaded successfully",
				slog.Int("success_count", len(uploadedImages)),
				slog.Int("failed_count", len(failedImages)))

			render.Status(r, http.StatusMultiStatus)
			render.JSON(w, r, PartialSuccessResponse{
				Response:     response.OK(),
				ImageIDs:     imageIDs,
				SuccessCount: len(uploadedImages),
				FailedCount:  len(failedImages),
				Failed:       failedImages,
			})
			return
		}

		log.Info("images uploaded successfully",
			slog.Int("success_count", len(uploadedImages)),
			slog.Int("total_count", len(files)))

		imageIDs := make([]string, len(uploadedImages))
		for i, img := range uploadedImages {
			imageIDs[i] = img.ImageID
		}

		render.JSON(w, r, Response{
			Response: response.OK(),
			ImageIDs: imageIDs,
		})
	}
}

func processImage(
	fileHeader *multipart.FileHeader,
	profileID, postID string,
	imageUploader ImageUploader,
	imageDBUploader ImageDBUploader,
	imageMeta *app_config.ImageMeta,
) (image.UploadImage, error) {
	const op = "handlers.upload_post_images.processImage"

	extension := strings.TrimPrefix(filepath.Ext(fileHeader.Filename), ".")
	if extension == "" {
		return image.UploadImage{}, fmt.Errorf("%s: file has no extension", op)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return image.UploadImage{}, fmt.Errorf("%s: failed to open file: %w", op, err)
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		return image.UploadImage{}, fmt.Errorf("%s: failed to read file: %w", op, err)
	}

	if len(imageData) > imageMeta.MaxImageSize<<20 {
		return image.UploadImage{}, fmt.Errorf("%s: file too large", op)
	}

	imageID := imageUploader.GenerateImageID()

	width, height, err := imageUploader.GetImageDimensions(imageData)
	if err != nil {
		return image.UploadImage{}, fmt.Errorf("%s: failed to get image dimensions: %w", op, err)
	}

	id, err := imageDBUploader.SaveImage(profileID, imageID, width, height, extension)
	if err != nil {
		return image.UploadImage{}, fmt.Errorf("%s: failed to save image to DB: %w", op, err)
	}

	err = imageDBUploader.SavePostImage(postID, id)
	if err != nil {
		return image.UploadImage{}, fmt.Errorf("%s: failed to save post image relation: %w", op, err)
	}

	fileName := imageID + filepath.Ext(fileHeader.Filename)
	err = imageUploader.SaveImageToDisk(imageData, imageMeta.ImageDirectory, fileName)
	if err != nil {
		return image.UploadImage{}, fmt.Errorf("%s failed to save image to disk: %w", op, err)
	}

	return image.UploadImage{
		ProfileID: profileID,
		ImageID:   imageID,
		Width:     width,
		Height:    height,
		Extension: extension,
	}, nil
}
