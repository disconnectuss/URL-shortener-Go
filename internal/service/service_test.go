package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"url-shortener/internal/model"
)

type mockStorage struct {
	urls map[string]string
}

func newMockStorage() *mockStorage {
	return &mockStorage{urls: make(map[string]string)}
}

func (m *mockStorage) Save(shortCode, originalURL string, expiresAt *time.Time) error {
	m.urls[shortCode] = originalURL
	return nil
}

func (m *mockStorage) Get(shortCode string) (string, error) {
	url, ok := m.urls[shortCode]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return url, nil
}

func (m *mockStorage) IncrementClick(shortCode string) error { return nil }

func (m *mockStorage) GetStats(shortCode string) (*model.URLStats, error) {
	url, ok := m.urls[shortCode]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return &model.URLStats{
		ShortCode:   shortCode,
		OriginalURL: url,
		ClickCount:  0,
		CreatedAt:   time.Now().String(),
	}, nil
}

func (m *mockStorage) CleanupExpired() (int64, error) { return 0, nil }
func (m *mockStorage) Close() error                   { return nil }

func TestShortenValid(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")

	resp, err := svc.Shorten("https://go.dev", "")
	if err != nil {
		t.Fatal("Shorten failed:", err)
	}
	if resp.ShortURL == "" {
		t.Error("short_url should not be empty")
	}
	if resp.ExpiresAt != nil {
		t.Error("expires_at should be nil when no expiry set")
	}
}

func TestShortenWithExpiry(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")

	resp, err := svc.Shorten("https://go.dev", "24h")
	if err != nil {
		t.Fatal("Shorten failed:", err)
	}
	if resp.ExpiresAt == nil {
		t.Error("expires_at should not be nil when expiry set")
	}
}

func TestShortenEmptyURL(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")

	_, err := svc.Shorten("", "")
	if err == nil {
		t.Error("expected error for empty URL")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestShortenInvalidURL(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")

	tests := []struct {
		name string
		url  string
	}{
		{"no scheme", "go.dev"},
		{"ftp scheme", "ftp://example.com"},
		{"missing host", "http://"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Shorten(tt.url, "")
			if err == nil {
				t.Errorf("expected error for URL %q", tt.url)
			}
			if !errors.Is(err, ErrValidation) {
				t.Errorf("expected ErrValidation, got: %v", err)
			}
		})
	}
}

func TestShortenInvalidExpiry(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")

	_, err := svc.Shorten("https://go.dev", "abc")
	if err == nil {
		t.Error("expected error for invalid expiry")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestResolve(t *testing.T) {
	store := newMockStorage()
	store.urls["abc12345"] = "https://go.dev"
	svc := New(store, nil, "http://localhost:8080")

	url, err := svc.Resolve("abc12345")
	if err != nil {
		t.Fatal("Resolve failed:", err)
	}
	if url != "https://go.dev" {
		t.Errorf("got %q, want %q", url, "https://go.dev")
	}
}

func TestResolveNotFound(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")

	_, err := svc.Resolve("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent code")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetStatsNotFound(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")

	_, err := svc.GetStats("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent code")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
