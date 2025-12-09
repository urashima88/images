package postgres

import (
	"database/sql"
	"fmt"
	app_config "images/internal/config/app-config"
	"images/internal/lib/api/image"
	"images/internal/lib/api/tag"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func New(cfg *app_config.Config) (*Storage, error) {
	const op = "storage.postgres.New"

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Db.Host,
		cfg.Db.Port,
		cfg.Db.User,
		cfg.Db.Password,
		cfg.Db.Name,
		cfg.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveImage(profileID, imageID string, width, height int, extension string, imageData []byte, imageDir, fileName string) (string, error) {
	const op = "storage.postgres.SaveImage"

	tx, err := s.db.Begin()
	if err != nil {
		return "", fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO images (profile_id, image_id, width, height, extension)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`

	var createdAt time.Time
	err = tx.QueryRow(query, profileID, imageID, width, height, extension).Scan(&createdAt)
	if err != nil {
		return "", fmt.Errorf("%s: failed to save image: %w", op, err)
	}

	filePath := filepath.Join(imageDir, fileName)
	if err := os.WriteFile(filePath, imageData, 0644); err != nil {
		return "", fmt.Errorf("%s: failed to write file: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return createdAt.Format(time.RFC3339), nil
}

func (s *Storage) GetImagesByIDs(imageIDs []string) ([]image.Image, error) {
	const op = "storage.postgres.GetImagesByPostID"

	if len(imageIDs) == 0 {
		return []image.Image{}, nil
	}

	query := `
		SELECT image_id, width, height, extension, created_at
		FROM images 
		WHERE image_id = ANY($1)
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, pq.Array(imageIDs))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query images by ids: %w", op, err)
	}
	defer rows.Close()

	var images []image.Image
	for rows.Next() {
		var img image.Image

		err := rows.Scan(
			&img.ImageID,
			&img.Width,
			&img.Height,
			&img.Extension,
			&img.CreatedAt,
		)

		if err != nil {
			slog.Error("failed to scan image row",
				slog.String("op", op),
				slog.String("error", err.Error()))
			continue
		}

		img.Tags = []tag.Tag{}
		images = append(images, img)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: error iterating image rows: %w", op, err)
	}

	return images, nil
}

func (s *Storage) CreateTags(tagNames []string) ([]tag.Tag, error) {
	const op = "storage.postgres.CreatePostTags"

	if len(tagNames) == 0 {
		return []tag.Tag{}, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	tempTableQuery := `
		CREATE TEMP TABLE temp_tags (
			name TEXT NOT NULL,
			tag_order INT NOT NULL
		) ON COMMIT DROP
	`

	_, err = tx.Exec(tempTableQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create temp table: %w", op, err)
	}

	for i, name := range tagNames {
		_, err := tx.Exec("INSERT INTO temp_tags (name, tag_order) VALUES ($1, $2)", name, i)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to insert into temp table: %w", op, err)
		}

	}
	insertQuery := `
		INSERT INTO tags (name)
		SELECT DISTINCT name FROM temp_tags
		ON CONFLICT (name) DO NOTHING
	`

	_, err = tx.Exec(insertQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to insert tags: %w", op, err)
	}

	selectQuery := `
		SELECT t.id, t.name, t.created_at
		FROM tags t
		JOIN temp_tags tt ON t.name = tt.name
		ORDER BY tt.tag_order
	`

	rows, err := tx.Query(selectQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query tags: %w", op, err)
	}
	defer rows.Close()

	var tags []tag.Tag
	for rows.Next() {
		var tag tag.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: failed to scan tag: %w", op, err)
		}
		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: error iterating rows: %w", op, err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return tags, nil
}

func (s *Storage) GetTagsByIDs(tagIDs []string) ([]tag.Tag, error) {
	const op = "storage.postgres.GetTagsByIDs"

	if len(tagIDs) == 0 {
		return []tag.Tag{}, nil
	}

	query := `
		SELECT id, name, created_at
		FROM tags
		WHERE id = ANY($1)
		ORDER BY name
	`

	rows, err := s.db.Query(query, pq.Array(tagIDs))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get tags by IDs: %w", op, err)
	}
	defer rows.Close()

	var tags []tag.Tag
	for rows.Next() {
		var tag tag.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt); err != nil {
			slog.Error("failed to scan tag row",
				slog.String("op", op),
				slog.String("error", err.Error()))
			continue
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: error iterating tag rows: %w", op, err)
	}

	return tags, nil
}
