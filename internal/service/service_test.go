package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

func (m *mockStorage) Save(_ context.Context, shortCode, originalURL string, expiresAt *time.Time) error {
	m.urls[shortCode] = originalURL
	return nil
}

func (m *mockStorage) Get(_ context.Context, shortCode string) (string, error) {
	url, ok := m.urls[shortCode]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return url, nil
}

func (m *mockStorage) IncrementClick(_ context.Context, shortCode string) error { return nil }

func (m *mockStorage) GetStats(_ context.Context, shortCode string) (*model.URLStats, error) {
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

func (m *mockStorage) Delete(_ context.Context, shortCode string) error {
	if _, ok := m.urls[shortCode]; !ok {
		return fmt.Errorf("not found")
	}
	delete(m.urls, shortCode)
	return nil
}

func (m *mockStorage) CleanupExpired(_ context.Context) (int64, error) { return 0, nil }
func (m *mockStorage) Close() error                                    { return nil }

func TestShortenValid(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")
	ctx := context.Background()

	resp, err := svc.Shorten(ctx, "https://go.dev", "", "")
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
	ctx := context.Background()

	resp, err := svc.Shorten(ctx, "https://go.dev", "24h", "")
	if err != nil {
		t.Fatal("Shorten failed:", err)
	}
	if resp.ExpiresAt == nil {
		t.Error("expires_at should not be nil when expiry set")
	}
}

func TestShortenEmptyURL(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")
	ctx := context.Background()

	_, err := svc.Shorten(ctx, "", "", "")
	if err == nil {
		t.Error("expected error for empty URL")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestShortenInvalidURL(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")
	ctx := context.Background()

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
			_, err := svc.Shorten(ctx, tt.url, "", "")
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
	ctx := context.Background()

	_, err := svc.Shorten(ctx, "https://go.dev", "abc", "")
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
	ctx := context.Background()

	url, err := svc.Resolve(ctx, "abc12345")
	if err != nil {
		t.Fatal("Resolve failed:", err)
	}
	if url != "https://go.dev" {
		t.Errorf("got %q, want %q", url, "https://go.dev")
	}
}

func TestResolveNotFound(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")
	ctx := context.Background()

	_, err := svc.Resolve(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent code")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetStatsNotFound(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")
	ctx := context.Background()

	_, err := svc.GetStats(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent code")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestShortenCustomCode(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")
	ctx := context.Background()

	resp, err := svc.Shorten(ctx, "https://go.dev", "", "my-link")
	if err != nil {
		t.Fatal("Shorten with custom code failed:", err)
	}
	if resp.ShortURL != "http://localhost:8080/my-link" {
		t.Errorf("got %q, want %q", resp.ShortURL, "http://localhost:8080/my-link")
	}
}

func TestShortenCustomCodeInvalid(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")
	ctx := context.Background()

	tests := []struct {
		name string
		code string
	}{
		{"too short", "ab"},
		{"has spaces", "my link"},
		{"special chars", "my@link!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Shorten(ctx, "https://go.dev", "", tt.code)
			if err == nil {
				t.Errorf("expected error for custom code %q", tt.code)
			}
			if !errors.Is(err, ErrValidation) {
				t.Errorf("expected ErrValidation, got: %v", err)
			}
		})
	}
}

func TestShortenURLTooLong(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")
	ctx := context.Background()

	longURL := "https://example.com/" + strings.Repeat("a", 2048)
	_, err := svc.Shorten(ctx, longURL, "", "")
	if err == nil {
		t.Error("expected error for too-long URL")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestDelete(t *testing.T) {
	store := newMockStorage()
	store.urls["abc12345"] = "https://go.dev"
	svc := New(store, nil, "http://localhost:8080")
	ctx := context.Background()

	if err := svc.Delete(ctx, "abc12345"); err != nil {
		t.Fatal("Delete failed:", err)
	}

	_, err := svc.Resolve(ctx, "abc12345")
	if !errors.Is(err, ErrNotFound) {
		t.Error("expected URL to be deleted")
	}
}

func TestDeleteNotFound(t *testing.T) {
	svc := New(newMockStorage(), nil, "http://localhost:8080")
	ctx := context.Background()

	err := svc.Delete(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent code")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
