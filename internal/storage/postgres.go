package storage

import (
	"database/sql"
	"time"

	"url-shortener/internal/model"

	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgres(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id           SERIAL PRIMARY KEY,
			short_code   TEXT UNIQUE NOT NULL,
			original_url TEXT NOT NULL,
			click_count  INTEGER DEFAULT 0,
			created_at   TIMESTAMPTZ DEFAULT NOW(),
			expires_at   TIMESTAMPTZ
		)
	`)
	if err != nil {
		return nil, err
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Save(shortCode, originalURL string, expiresAt *time.Time) error {
	_, err := s.db.Exec(
		"INSERT INTO urls (short_code, original_url, expires_at) VALUES ($1, $2, $3)",
		shortCode, originalURL, expiresAt,
	)
	return err
}

func (s *PostgresStore) Get(shortCode string) (string, error) {
	var originalURL string
	err := s.db.QueryRow(
		"SELECT original_url FROM urls WHERE short_code = $1 AND (expires_at IS NULL OR expires_at > NOW())",
		shortCode,
	).Scan(&originalURL)
	return originalURL, err
}

func (s *PostgresStore) IncrementClick(shortCode string) error {
	_, err := s.db.Exec(
		"UPDATE urls SET click_count = click_count + 1 WHERE short_code = $1",
		shortCode,
	)
	return err
}

func (s *PostgresStore) GetStats(shortCode string) (*model.URLStats, error) {
	var stats model.URLStats
	err := s.db.QueryRow(
		"SELECT short_code, original_url, click_count, created_at, expires_at FROM urls WHERE short_code = $1",
		shortCode,
	).Scan(&stats.ShortCode, &stats.OriginalURL, &stats.ClickCount, &stats.CreatedAt, &stats.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *PostgresStore) CleanupExpired() (int64, error) {
	result, err := s.db.Exec("DELETE FROM urls WHERE expires_at IS NOT NULL AND expires_at <= NOW()")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}
