package postgres

import (
	"database/sql"
	"fmt"
	app_config "images/internal/config/app-config"
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
