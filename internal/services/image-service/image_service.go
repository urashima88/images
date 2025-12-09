package image_service

import (
	"bytes"
	"fmt"
	"image"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/google/uuid"
)

type ImageService struct {
	BaseURL string
}

func New(baseURL string) *ImageService {
	return &ImageService{BaseURL: baseURL}
}

func (i *ImageService) GenerateImageID() string {
	return uuid.New().String()
}

func (i *ImageService) GetImageDimensions(imageData []byte) (width, height int, err error) {
	const op = "services.image_service.GetImageDimensions"

	reader := bytes.NewReader(imageData)
	config, _, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, fmt.Errorf("%s: failed to decode image: %w", op, err)
	}

	return config.Width, config.Height, nil
}

func (i *ImageService) GetImageURL(imageID, extension string) string {
	if i.BaseURL == "" {
		return ""
	}

	return fmt.Sprintf(i.BaseURL + imageID + "." + extension)
}
