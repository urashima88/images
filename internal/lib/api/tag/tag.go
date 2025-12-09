package tag

import "time"

type Tag struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type UploadTagsData struct {
	Tags map[string][]string `json:"tags"`
}
