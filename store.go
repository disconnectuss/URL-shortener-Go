package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type URLStore struct {
	db *sql.DB
}

type URLStats struct {
	ShortCode   string  `json:"short_code"`
	OriginalURL string  `json:"original_url"`
	ClickCount  int     `json:"click_count"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
}

func NewURLStore(dbPath string) (*URLStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
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

	return &URLStore{db: db}, nil
}

func (s *URLStore) Save(shortCode, originalURL string, expiresAt *time.Time) error {
	var expiresUTC *string
	if expiresAt != nil {
		s := expiresAt.UTC().Format("2006-01-02 15:04:05")
		expiresUTC = &s
	}
	_, err := s.db.Exec(
		"INSERT INTO urls (short_code, original_url, expires_at) VALUES (?, ?, ?)",
		shortCode, originalURL, expiresUTC,
	)
	return err
}

func (s *URLStore) Get(shortCode string) (string, error) {
	var originalURL string
	err := s.db.QueryRow(
		"SELECT original_url FROM urls WHERE short_code = ? AND (expires_at IS NULL OR expires_at > datetime('now'))",
		shortCode,
	).Scan(&originalURL)
	return originalURL, err
}

func (s *URLStore) IncrementClick(shortCode string) error {
	_, err := s.db.Exec(
		"UPDATE urls SET click_count = click_count + 1 WHERE short_code = ?",
		shortCode,
	)
	return err
}

func (s *URLStore) GetStats(shortCode string) (*URLStats, error) {
	var stats URLStats
	err := s.db.QueryRow(
		"SELECT short_code, original_url, click_count, created_at, expires_at FROM urls WHERE short_code = ?",
		shortCode,
	).Scan(&stats.ShortCode, &stats.OriginalURL, &stats.ClickCount, &stats.CreatedAt, &stats.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *URLStore) CleanupExpired(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		result, err := s.db.Exec("DELETE FROM urls WHERE expires_at IS NOT NULL AND expires_at <= datetime('now')")
		if err != nil {
			log.Println("cleanup error:", err)
			continue
		}
		if count, _ := result.RowsAffected(); count > 0 {
			log.Printf("cleaned up %d expired URLs", count)
		}
	}
}

func (s *URLStore) Close() error {
	return s.db.Close()
}
