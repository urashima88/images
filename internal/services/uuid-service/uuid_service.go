package uuid_service

import (
	"strings"

	"github.com/google/uuid"
)

type UUIDService struct {
}

func New() *UUIDService {
	return &UUIDService{}
}

func (i *UUIDService) CleanAndValidateIDs(ids []string) []string {
	var cleaned []string
	seen := make(map[string]bool)

	for _, id := range ids {
		id := strings.TrimSpace(id)
		if _, err := uuid.Parse(id); err != nil {
			continue
		}

		if !seen[id] {
			seen[id] = true
			cleaned = append(cleaned, id)
		}
	}
	return cleaned
}
