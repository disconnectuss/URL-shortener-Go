package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"url-shortener/internal/cache"
	"url-shortener/internal/model"
	"url-shortener/internal/storage"
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

func (s *URLService) Shorten(rawURL, expiresIn string) (*model.ShortenResponse, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("url field is required")
	}

	if err := validateURL(rawURL); err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if expiresIn != "" {
		d, err := parseDuration(expiresIn)
		if err != nil {
			return nil, fmt.Errorf("invalid expires_in: use format like 30m, 24h, 7d")
		}
		t := time.Now().Add(d)
		expiresAt = &t
	}

	code, err := generateShortCode()
	if err != nil {
		return nil, fmt.Errorf("could not generate short code")
	}

	if err := s.store.Save(code, rawURL, expiresAt); err != nil {
		return nil, fmt.Errorf("could not save URL: %w", err)
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

func (s *URLService) Resolve(shortCode string) (string, error) {
	if s.cache != nil {
		if url, err := s.cache.Get(shortCode); err == nil {
			s.store.IncrementClick(shortCode)
			return url, nil
		}
	}

	originalURL, err := s.store.Get(shortCode)
	if err != nil {
		return "", fmt.Errorf("URL not found")
	}

	if s.cache != nil {
		s.cache.Set(shortCode, originalURL)
	}

	s.store.IncrementClick(shortCode)
	return originalURL, nil
}

func (s *URLService) GetStats(shortCode string) (*model.URLStats, error) {
	return s.store.GetStats(shortCode)
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
	bytes := make([]byte, 3)
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
