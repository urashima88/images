package image

import "time"

type UploadImage struct {
	ProfileID string
	ImageID   string
	Width     int
	Height    int
	Extension string
}

type FailedImageResponse struct {
	FileName string `json:"file_name"`
	Error    string `json:"error"`
}

type DownloadImageResponse struct {
	ImageID   string    `json:"image_id"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Score     int       `json:"score"`
	Extension string    `json:"extension"`
	CreatedAt time.Time `json:"created_at"`
	FileURL   string    `json:"file_url,omitempty"`
}
