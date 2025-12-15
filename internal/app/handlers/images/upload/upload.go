package images_upload

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
	Images []image.Image `json:"images"`
}

type PartialSuccessResponse struct {
	response.Response
	Images       []image.Image               `json:"images"`
	SuccessCount int                         `json:"success_count"`
	FailedCount  int                         `json:"failed_count"`
	Failed       []image.FailedImageResponse `json:"failed"`
}

type ImageUploader interface {
	GenerateImageID() string
	GetImageDimensions(imageData []byte) (int, int, error)
	GetImageURL(imageID, extension string) string
}

type ImageDBUploader interface {
	SaveImage(profileID, imageID string, width, height int, extension string, imageData []byte, imageDir, fileName string) (string, error)
}

const (
	errNoExtension   = "file has no extension"
	errFileTooLarge  = "file too large"
	errInvalidFormat = "invalid image format"
	errSaveFailed    = "failed to save image"
	errOpenFailed    = "failed to open file"
	errReadFailed    = "failed to read file"
)

// @Summary Upload images
// @Description
// Allows to upload from 1 to 10 images in one request.
// Supported formats: JPEG, PNG, GIF
// Maximum size of a single file: 20 MB.
// @Tags Images
// @Accept multipart/form-data
// @Produce json
// @Param X-Profile-ID header string true "User profile UUID (v4 format)"
// @Param images formData file true "Image files to download"
// @Success 200 {object} Response "All images have been uploaded successfully"
// @Success 207 {object} PartialSuccessResponse "Some of the images were uploaded successfully, some with errors"
// @Failure 400 {object} response.Response "Validation error: missing header, invalid UUID, no files, too many files"
// @Failure 422 {object} response.Response "None of the images could be processed"
// @Failure 500 {object} response.Response "Internal server error"
// @Security X-Profile-ID
// @Router /images [post]
func New(log *slog.Logger, imageUploader ImageUploader, imageDBUploader ImageDBUploader, imageMeta *app_config.ImageMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.images.upload.New"

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
			log.Error("invalid profile id format",
				slog.String("profile_id", profileID),
				slog.String("error", err.Error()))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid profile id format"))
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

		if len(files) > imageMeta.UploadNumberImages {
			log.Error("too many images", slog.Int("count", len(files)))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error(fmt.Sprintf("maximum %d images allowed", imageMeta.UploadNumberImages)))
			return
		}

		log.Info("processing images", slog.Int("image_count", len(files)))

		var uploadedImages []image.Image
		var failedImages []image.FailedImageResponse

		for i, fileHeader := range files {
			img, errMsg, err := processImage(fileHeader, profileID, imageUploader, imageDBUploader, imageMeta)
			if err != nil {
				failedImages = append(failedImages, image.FailedImageResponse{
					FileName: fileHeader.Filename,
					Error:    errMsg,
				})

				log.Error("failed to process image",
					slog.Int("index", i),
					slog.String("filename", fileHeader.Filename),
					slog.String("error", err.Error()))
				continue
			}
			uploadedImages = append(uploadedImages, *img)
		}

		if len(uploadedImages) == 0 {
			log.Error("all images failed to process")
			render.Status(r, http.StatusUnprocessableEntity)
			render.JSON(w, r, response.Error("failed to process any image"))
			return
		}

		if len(failedImages) > 0 {
			log.Info("only a part of images uploaded successfully",
				slog.Int("success_count", len(uploadedImages)),
				slog.Int("failed_count", len(failedImages)))

			render.Status(r, http.StatusMultiStatus)
			render.JSON(w, r, PartialSuccessResponse{
				Response:     response.OK(),
				Images:       uploadedImages,
				SuccessCount: len(uploadedImages),
				FailedCount:  len(failedImages),
				Failed:       failedImages,
			})
			return
		}

		log.Info("images uploaded successfully",
			slog.Int("success_count", len(uploadedImages)),
			slog.Int("total_count", len(files)))

		render.JSON(w, r, Response{
			Response: response.OK(),
			Images:   uploadedImages,
		})
	}
}

func processImage(
	fileHeader *multipart.FileHeader,
	profileID string,
	imageUploader ImageUploader,
	imageDBUploader ImageDBUploader,
	imageMeta *app_config.ImageMeta,
) (*image.Image, string, error) {
	const op = "handlers.images.upload.processImage"

	extension := strings.TrimPrefix(filepath.Ext(fileHeader.Filename), ".")
	if extension == "" {
		return nil, errNoExtension, fmt.Errorf("%s: file has no extension", op)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, errOpenFailed, fmt.Errorf("%s: failed to open file: %w", op, err)
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		return nil, errReadFailed, fmt.Errorf("%s: failed to read file: %w", op, err)
	}

	if len(imageData) > imageMeta.MaxImageSize<<20 {
		return nil, errFileTooLarge, fmt.Errorf("%s: file size %d exceeds limit %d", op, len(imageData), imageMeta.MaxImageSize<<20)
	}

	imageID := imageUploader.GenerateImageID()

	width, height, err := imageUploader.GetImageDimensions(imageData)
	if err != nil {
		return nil, errInvalidFormat, fmt.Errorf("%s: failed to get image dimensions: %w", op, err)
	}

	fileName := imageID + filepath.Ext(fileHeader.Filename)
	createdAt, err := imageDBUploader.SaveImage(profileID, imageID, width, height, extension, imageData, imageMeta.ImageDirectory, fileName)
	if err != nil {
		return nil, errSaveFailed, fmt.Errorf("%s: failed to save image to DB: %w", op, err)
	}

	return &image.Image{
		ImageID:   imageID,
		Width:     width,
		Height:    height,
		Extension: extension,
		CreatedAt: createdAt,
		FileURL:   imageUploader.GetImageURL(imageID, extension),
	}, "", nil
}
