package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"url-shortener/internal/cache"
	"url-shortener/internal/model"
	"url-shortener/internal/storage"
)

var (
	ErrValidation = errors.New("validation error")
	ErrInternal   = errors.New("internal error")
	ErrNotFound   = errors.New("not found")
)

type URLService struct {
	store   storage.Storage
	cache   *cache.Cache
	baseURL string
}

func New(store storage.Storage, cache *cache.Cache, baseURL string) *URLService {
	return &URLService{
		store:   store,
		cache:   cache,
		baseURL: baseURL,
	}
}

func (s *URLService) Shorten(ctx context.Context, rawURL, expiresIn string) (*model.ShortenResponse, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("%w: url field is required", ErrValidation)
	}

	if err := validateURL(rawURL); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}

	var expiresAt *time.Time
	if expiresIn != "" {
		d, err := parseDuration(expiresIn)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid expires_in: use format like 30m, 24h, 7d", ErrValidation)
		}
		t := time.Now().Add(d)
		expiresAt = &t
	}

	var code string
	for range 3 {
		var err error
		code, err = generateShortCode()
		if err != nil {
			return nil, fmt.Errorf("%w: could not generate short code", ErrInternal)
		}

		err = s.store.Save(ctx, code, rawURL, expiresAt)
		if err == nil {
			break
		}

		if code != "" {
			continue
		}
		return nil, fmt.Errorf("%w: could not save URL", ErrInternal)
	}

	resp := &model.ShortenResponse{
		ShortURL: fmt.Sprintf("%s/%s", s.baseURL, code),
	}
	if expiresAt != nil {
		formatted := expiresAt.UTC().Format(time.RFC3339)
		resp.ExpiresAt = &formatted
	}

	return resp, nil
}

func (s *URLService) Resolve(ctx context.Context, shortCode string) (string, error) {
	if s.cache != nil {
		if cachedURL, err := s.cache.Get(shortCode); err == nil {
			if _, err := s.store.Get(ctx, shortCode); err == nil {
				if err := s.store.IncrementClick(ctx, shortCode); err != nil {
					slog.Error("increment click failed", "error", err, "short_code", shortCode)
				}
				return cachedURL, nil
			}
			s.cache.Delete(shortCode)
		}
	}

	originalURL, err := s.store.Get(ctx, shortCode)
	if err != nil {
		return "", fmt.Errorf("%w: URL not found", ErrNotFound)
	}

	if s.cache != nil {
		if err := s.cache.Set(shortCode, originalURL); err != nil {
			slog.Warn("cache set failed", "error", err, "short_code", shortCode)
		}
	}

	if err := s.store.IncrementClick(ctx, shortCode); err != nil {
		slog.Error("increment click failed", "error", err, "short_code", shortCode)
	}
	return originalURL, nil
}

func (s *URLService) GetStats(ctx context.Context, shortCode string) (*model.URLStats, error) {
	stats, err := s.store.GetStats(ctx, shortCode)
	if err != nil {
		return nil, fmt.Errorf("%w: URL not found", ErrNotFound)
	}
	return stats, nil
}

func validateURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("invalid URL: must start with http:// or https://")
	}
	if parsed.Host == "" {
		return fmt.Errorf("invalid URL: missing host")
	}
	return nil
}

func generateShortCode() (string, error) {
	bytes := make([]byte, 4) // 4 byte = 8 hex char = 4.2 billion combinations
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func parseDuration(s string) (time.Duration, error) {
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
