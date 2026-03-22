package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type dbDialect struct {
	placeholder func(n int) string
	now         string
	createTable string
}

var sqliteDialect = dbDialect{
	placeholder: func(n int) string { return "?" },
	now:         "datetime('now')",
	createTable: `
		CREATE TABLE IF NOT EXISTS urls (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			short_code   TEXT UNIQUE NOT NULL,
			original_url TEXT NOT NULL,
			click_count  INTEGER DEFAULT 0,
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at   DATETIME
		)`,
}

var postgresDialect = dbDialect{
	placeholder: func(n int) string { return fmt.Sprintf("$%d", n) },
	now:         "NOW()",
	createTable: `
		CREATE TABLE IF NOT EXISTS urls (
			id           SERIAL PRIMARY KEY,
			short_code   TEXT UNIQUE NOT NULL,
			original_url TEXT NOT NULL,
			click_count  INTEGER DEFAULT 0,
			created_at   TIMESTAMPTZ DEFAULT NOW(),
			expires_at   TIMESTAMPTZ
		)`,
}

type URLStore struct {
	db      *sql.DB
	dialect dbDialect
}

type URLStats struct {
	ShortCode   string  `json:"short_code"`
	OriginalURL string  `json:"original_url"`
	ClickCount  int     `json:"click_count"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
}

func NewURLStore(driver, dsn string) (*URLStore, error) {
	var dialect dbDialect
	switch driver {
	case "sqlite3":
		dialect = sqliteDialect
	case "postgres":
		dialect = postgresDialect
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	_, err = db.Exec(dialect.createTable)
	if err != nil {
		return nil, err
	}

	return &URLStore{db: db, dialect: dialect}, nil
}

func (s *URLStore) Save(shortCode, originalURL string, expiresAt *time.Time) error {
	p := s.dialect.placeholder
	var expiresVal any
	if expiresAt != nil {
		expiresVal = expiresAt.UTC().Format("2006-01-02 15:04:05")
	}
	_, err := s.db.Exec(
		fmt.Sprintf("INSERT INTO urls (short_code, original_url, expires_at) VALUES (%s, %s, %s)", p(1), p(2), p(3)),
		shortCode, originalURL, expiresVal,
	)
	return err
}

func (s *URLStore) Get(shortCode string) (string, error) {
	var originalURL string
	p := s.dialect.placeholder
	query := fmt.Sprintf(
		"SELECT original_url FROM urls WHERE short_code = %s AND (expires_at IS NULL OR expires_at > %s)",
		p(1), s.dialect.now,
	)
	err := s.db.QueryRow(query, shortCode).Scan(&originalURL)
	return originalURL, err
}

func (s *URLStore) IncrementClick(shortCode string) error {
	p := s.dialect.placeholder
	_, err := s.db.Exec(
		fmt.Sprintf("UPDATE urls SET click_count = click_count + 1 WHERE short_code = %s", p(1)),
		shortCode,
	)
	return err
}

func (s *URLStore) GetStats(shortCode string) (*URLStats, error) {
	var stats URLStats
	p := s.dialect.placeholder
	err := s.db.QueryRow(
		fmt.Sprintf("SELECT short_code, original_url, click_count, created_at, expires_at FROM urls WHERE short_code = %s", p(1)),
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
		query := fmt.Sprintf("DELETE FROM urls WHERE expires_at IS NOT NULL AND expires_at <= %s", s.dialect.now)
		result, err := s.db.Exec(query)
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
