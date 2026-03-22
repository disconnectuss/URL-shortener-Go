package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type URLStore struct {
	db *sql.DB
}

type URLStats struct {
	ShortCode   string `json:"short_code"`
	OriginalURL string `json:"original_url"`
	ClickCount  int    `json:"click_count"`
	CreatedAt   string `json:"created_at"`
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
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, err
	}

	return &URLStore{db: db}, nil
}

func (s *URLStore) Save(shortCode, originalURL string) error {
	_, err := s.db.Exec(
		"INSERT INTO urls (short_code, original_url) VALUES (?, ?)",
		shortCode, originalURL,
	)
	return err
}

func (s *URLStore) Get(shortCode string) (string, error) {
	var originalURL string
	err := s.db.QueryRow(
		"SELECT original_url FROM urls WHERE short_code = ?",
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
		"SELECT short_code, original_url, click_count, created_at FROM urls WHERE short_code = ?",
		shortCode,
	).Scan(&stats.ShortCode, &stats.OriginalURL, &stats.ClickCount, &stats.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *URLStore) Close() error {
	return s.db.Close()
}
