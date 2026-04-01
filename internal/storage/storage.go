package storage

import (
	"context"
	"time"

	"url-shortener/internal/model"
)

type Storage interface {
	Save(ctx context.Context, shortCode, originalURL string, expiresAt *time.Time) error
	Get(ctx context.Context, shortCode string) (string, error)
	IncrementClick(ctx context.Context, shortCode string) error
	GetStats(ctx context.Context, shortCode string) (*model.URLStats, error)
	CleanupExpired(ctx context.Context) (int64, error)
	Close() error
}
