package storage

import (
	"context"
	"database/sql"
	"time"

	"url-shortener/internal/model"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLite(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			short_code   TEXT UNIQUE NOT NULL,
			original_url TEXT NOT NULL,
			click_count  INTEGER DEFAULT 0,
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at   DATETIME
		)
	`)
	if err != nil {
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Save(ctx context.Context, shortCode, originalURL string, expiresAt *time.Time) error {
	var expiresVal any
	if expiresAt != nil {
		expiresVal = expiresAt.UTC().Format("2006-01-02 15:04:05")
	}
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO urls (short_code, original_url, expires_at) VALUES (?, ?, ?)",
		shortCode, originalURL, expiresVal,
	)
	return err
}

func (s *SQLiteStore) Get(ctx context.Context, shortCode string) (string, error) {
	var originalURL string
	err := s.db.QueryRowContext(ctx,
		"SELECT original_url FROM urls WHERE short_code = ? AND (expires_at IS NULL OR expires_at > datetime('now'))",
		shortCode,
	).Scan(&originalURL)
	return originalURL, err
}

func (s *SQLiteStore) Delete(ctx context.Context, shortCode string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM urls WHERE short_code = ?", shortCode)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) IncrementClick(ctx context.Context, shortCode string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE urls SET click_count = click_count + 1 WHERE short_code = ?",
		shortCode,
	)
	return err
}

func (s *SQLiteStore) GetStats(ctx context.Context, shortCode string) (*model.URLStats, error) {
	var stats model.URLStats
	err := s.db.QueryRowContext(ctx,
		"SELECT short_code, original_url, click_count, created_at, expires_at FROM urls WHERE short_code = ?",
		shortCode,
	).Scan(&stats.ShortCode, &stats.OriginalURL, &stats.ClickCount, &stats.CreatedAt, &stats.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *SQLiteStore) CleanupExpired(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, "DELETE FROM urls WHERE expires_at IS NOT NULL AND expires_at <= datetime('now')")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
