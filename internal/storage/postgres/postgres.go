package postgres

import (
	"database/sql"
	"fmt"
	app_config "images/internal/config/app-config"
	"images/internal/lib/api/image"
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

		images = append(images, img)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: error iterating image rows: %w", op, err)
	}

	return images, nil
}
