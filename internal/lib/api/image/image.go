package image

import "time"

type UploadImage struct {
	ProfileID string
	ImageID   string
	Width     int
	Height    int
	Extension string
}

type ProcessImageError struct {
	Root     error
	Internal error
}

type FailedImageResponse struct {
	FileName string `json:"file_name"`
	Error    string `json:"error"`
}

type ImageInfoResponse struct {
	ImageID   string    `json:"image_id"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Extension string    `json:"extension"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	FileURL   string    `json:"file_url,omitempty"`
}
