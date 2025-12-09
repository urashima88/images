package image

import (
	"images/internal/lib/api/tag"
)

type Image struct {
	ImageID   string    `json:"image_id"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Extension string    `json:"extension"`
	Tags      []tag.Tag `json:"tags"`
	CreatedAt string    `json:"created_at"`
	FileURL   string    `json:"file_url,omitempty"`
}

type FailedImageResponse struct {
	FileName string `json:"file_name"`
	Error    string `json:"error"`
}

type ImageWithTags struct {
	ImageID string   `json:"image_id"`
	TagIDs  []string `json:"tag_ids,omitempty"`
}
