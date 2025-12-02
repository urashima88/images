package postgres

import (
	"database/sql"
	"fmt"
	app_config "images/internal/config/app-config"
	"images/internal/lib/api/image"
	"images/internal/lib/api/tag"
	"log/slog"

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

func (s *Storage) SaveImage(profileID, imageID string, width, height int, extension string) error {
	const op = "storage.postgres.SaveImage"

	query := `
		INSERT INTO images (profile_id, image_id, width, height, extension)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := s.db.Exec(query, profileID, imageID, width, height, extension)
	if err != nil {
		return fmt.Errorf("%s: failed to insert into images table: %w", op, err)
	}

	return nil
}

func (s *Storage) GetImagesByIDs(imageIDs []string) ([]image.DownloadImageResponse, error) {
	const op = "storage.postgres.GetImagesByPostID"

	query := `
		SELECT i.image_id, i.width, i.height, i.extension, i.score, i.created_at,
		COALESCE (
			ARRAY_AGG(DISTINCT t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL), '{}'::text[]
		) AS tags
		FROM images i
		LEFT JOIN image_tags it ON i.id = it.image_id
		LEFT JOIN tags t ON it.tag_id = t.id
		WHERE i.image_id = ANY($1)
		GROUP BY i.id, i.image_id, i.width, i.height, i.extension, i.score, i.created_at
		ORDER BY i.created_at ASC
	`

	rows, err := s.db.Query(query, pq.Array(imageIDs))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query images by ids: %w", op, err)
	}
	defer rows.Close()

	var images []image.DownloadImageResponse
	for rows.Next() {
		var img image.DownloadImageResponse
		var tags []string

		err := rows.Scan(
			&img.ImageID,
			&img.Width,
			&img.Height,
			&img.Extension,
			&img.Score,
			&img.CreatedAt,
			pq.Array(&tags),
		)
		if err != nil {
			slog.Error("failed to scan image row",
				slog.String("op", op),
				slog.String("error", err.Error()))
			continue
		}
		img.Tags = tags
		images = append(images, img)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: error iterating rows: %w", op, err)
	}
	return images, nil
}

func (s *Storage) ValidateImageOwnership(profileID, imageID string) (bool, error) {
	const op = "storage.postgres.ValidateImageOwnership"

	query := `
		SELECT EXISTS(
			SELECT 1 FROM images
			WHERE image_id = $1 AND profile_id = $2
		)
	`

	var exists bool
	err := s.db.QueryRow(query, imageID, profileID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: failed to validate ownership: %w", op, err)
	}

	return exists, nil
}

func (s *Storage) UpdateImageTags(imageID string, tagNames []string) ([]tag.Tag, error) {
	const op = "storage.postgres.UpdateImageTags"

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer tx.Rollback()

	var id string
	err = tx.QueryRow("SELECT id FROM images WHERE image_id = $1", imageID).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("%s: image not found: %w", op, err)
	}

	tagIDs, err := s.GetOrSaveTags(tx, tagNames)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get/save tags: %w", op, err)
	}

	err = s.UpdateImageTagsRelations(tx, id, tagIDs)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to update image-tag relations: %w", op, err)
	}

	createdTags, err := s.GetTagsByIDs(tx, tagIDs)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get created tags: %w", op, err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}
	return createdTags, nil
}

func (s *Storage) GetOrSaveTags(tx *sql.Tx, tagNames []string) ([]string, error) {
	const op = "storage.postgres.GetOrSaveTags"

	if len(tagNames) == 0 {
		return []string{}, nil
	}

	_, err := tx.Exec(`
		CREATE TEMP TABLE temp_tags (name TEXT) ON COMMIT DROP
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create temp table: %w", op, err)
	}

	stmt, err := tx.Prepare(pq.CopyIn("temp_tags", "name"))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to prepare copy statement: %w", op, err)
	}

	for _, name := range tagNames {
		_, err = stmt.Exec(name)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to insert tag %s into temp table: %w", op, name, err)
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to insert tags into temp table: %w", op, err)
	}
	stmt.Close()

	query := `
			WITH inserted_tags AS (
				INSERT INTO tags (name)
				SELECT DISTINCT name FROM temp_tags
				ON CONFLICT (name) DO NOTHING
				RETURNING id, name
			)
			SELECT id FROM inserted_tags
			UNION ALL
			SELECT t.id FROM tags t
			INNER JOIN temp_tags tt 
			ON t.name = tt.name
	`

	rows, err := tx.Query(query)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to upsert into tags table: %w", op, err)
	}
	defer rows.Close()

	var tagIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("%s: failed to scan tag id: %w", op, err)
		}
		tagIDs = append(tagIDs, id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: error iterating rows: %w", op, err)
	}

	return tagIDs, nil
}

func (s *Storage) UpdateImageTagsRelations(tx *sql.Tx, imageID string, tagIDs []string) error {
	const op = "storage.postgres.UpdateImageTagsRelations"

	deleteQuery := `DELETE FROM image_tags WHERE image_id = $1`
	_, err := tx.Exec(deleteQuery, imageID)
	if err != nil {
		return fmt.Errorf("%s: failed to delete old tags: %w", op, err)
	}

	if len(tagIDs) == 0 {
		return nil
	}

	insertQuery := `
		INSERT INTO image_tags (image_id, tag_id)
		SELECT $1, unnest($2::uuid[])
	`

	_, err = tx.Exec(insertQuery, imageID, pq.Array(tagIDs))
	if err != nil {
		return fmt.Errorf("%s: failed to insert into new tags: %w", op, err)
	}
	return nil
}

func (s *Storage) GetTagsByIDs(tx *sql.Tx, tagIDs []string) ([]tag.Tag, error) {
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

	rows, err := tx.Query(query, pq.Array(tagIDs))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get tags by IDs: %w", op, err)
	}
	defer rows.Close()

	var tags []tag.Tag
	for rows.Next() {
		var t tag.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: error scan tag: %w", op, err)
		}
		tags = append(tags, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: error iterating rows: %w", op, err)
	}

	return tags, nil
}

func (s *Storage) UpdateImageScore(imageID string, value int) (int, error) {
	const op = "storage.postgres.UpdateImageScore"

	query := `
		UPDATE images
		SET
			score = score + $1,
			updated_at = NOW()
		WHERE image_id = $2
		RETURNING score
	`
	var newScore int
	err := s.db.QueryRow(query, value, imageID).Scan(&newScore)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("%s: image_id=%s not found", op, imageID)
		}
		return 0, fmt.Errorf("%s: failed to update image score: %w", op, err)
	}
	return newScore, nil
}
