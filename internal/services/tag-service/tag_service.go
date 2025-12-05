package tag_service

import "strings"

type TagService struct {
	MaxTagLength   int
	ForbiddenChars []string
}

func New(maxTagLength int) *TagService {
	return &TagService{
		MaxTagLength: maxTagLength,
		ForbiddenChars: []string{
			"'", "\"", ";", "--", "/*", "*/", "#",
			"|", "&", "`", "$", "(", ")", "<", ">",
			"\n", "\r", "\t", "\x00",
			"!", "@", "%", "*", "+", "=", "[", "]", "{", "}",
			"\\", "/", "?", ":",
		},
	}
}

func (t *TagService) CleanAndValidateTags(tags []string) []string {
	var cleaned []string
	seen := make(map[string]bool)

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		tag = strings.ToLower(tag)

		tag = strings.Join(strings.Fields(tag), " ")

		if tag == "" {
			continue
		}

		if len(tag) > t.MaxTagLength {
			tag = tag[:t.MaxTagLength]
		}

		continueTagLoop := false

		for _, char := range t.ForbiddenChars {
			if strings.Contains(tag, char) {
				continueTagLoop = true
				break
			}
		}
		if continueTagLoop {
			continue
		}

		if !seen[tag] {
			seen[tag] = true
			cleaned = append(cleaned, tag)
		}
	}
	return cleaned
}
