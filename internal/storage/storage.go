package storage

import (
	"time"

	"url-shortener/internal/model"
)

type Storage interface {
	Save(shortCode, originalURL string, expiresAt *time.Time) error
	Get(shortCode string) (string, error)
	IncrementClick(shortCode string) error
	GetStats(shortCode string) (*model.URLStats, error)
	CleanupExpired() (int64, error)
	Close() error
}
