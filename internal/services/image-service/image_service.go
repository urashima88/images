package image_service

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

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

func (i *ImageService) SaveImageToDisk(imageData []byte, imageDir, fileName string) error {
	const op = "services.image_service.SaveImageToDisk"

	filePath := filepath.Join(imageDir, fileName)

	err := os.WriteFile(filePath, imageData, 0644)
	if err != nil {
		return fmt.Errorf("%s: failed to write file: %w", op, err)
	}

	return nil
}

func (i *ImageService) CleanImageIDs(imageIDs []string) []string {
	var cleaned []string
	seen := make(map[string]bool)

	for _, id := range imageIDs {
		id := strings.TrimSpace(id)
		if id == "" {
			continue
		}

		if !seen[id] {
			seen[id] = true
			cleaned = append(cleaned, id)
		}
	}
	return cleaned
}

func (i *ImageService) GetImageURL(imageID, extension string) string {
	return fmt.Sprintf(i.BaseURL + imageID + "." + extension)
}
