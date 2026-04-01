package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"url-shortener/internal/model"
	"url-shortener/internal/service"
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
		ClickCount:  5,
		CreatedAt:   "2026-01-01 00:00:00",
	}, nil
}

func (m *mockStorage) CleanupExpired() (int64, error) { return 0, nil }
func (m *mockStorage) Close() error                   { return nil }

func newTestService(store *mockStorage) *service.URLService {
	return service.New(store, nil, "http://localhost:8080")
}

func TestHandleShorten_Success(t *testing.T) {
	svc := newTestService(newMockStorage())
	handler := handleShorten(svc)

	body := `{"url": "https://go.dev"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var resp model.ShortenResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal("response JSON parse hatası:", err)
	}

	if resp.ShortURL == "" {
		t.Error("short_url boş olmamalı")
	}
}

func TestHandleShorten_InvalidJSON(t *testing.T) {
	svc := newTestService(newMockStorage())
	handler := handleShorten(svc)

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleShorten_EmptyURL(t *testing.T) {
	svc := newTestService(newMockStorage())
	handler := handleShorten(svc)

	body := `{"url": ""}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleShorten_InvalidURL(t *testing.T) {
	svc := newTestService(newMockStorage())
	handler := handleShorten(svc)

	body := `{"url": "not-a-url"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleRedirect_Success(t *testing.T) {
	store := newMockStorage()
	store.urls["abc12345"] = "https://go.dev"
	svc := newTestService(store)
	handler := handleRedirect(svc)

	req := httptest.NewRequest(http.MethodGet, "/abc12345", nil)
	req.SetPathValue("shortCode", "abc12345")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}

	location := rec.Header().Get("Location")
	if location != "https://go.dev" {
		t.Errorf("Location = %q, want %q", location, "https://go.dev")
	}
}

func TestHandleRedirect_NotFound(t *testing.T) {
	svc := newTestService(newMockStorage())
	handler := handleRedirect(svc)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	req.SetPathValue("shortCode", "nonexistent")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleStats_Success(t *testing.T) {
	store := newMockStorage()
	store.urls["abc12345"] = "https://go.dev"
	svc := newTestService(store)
	handler := handleStats(svc)

	req := httptest.NewRequest(http.MethodGet, "/stats/abc12345", nil)
	req.SetPathValue("shortCode", "abc12345")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var stats model.URLStats
	if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
		t.Fatal("response JSON parse hatası:", err)
	}

	if stats.OriginalURL != "https://go.dev" {
		t.Errorf("OriginalURL = %q, want %q", stats.OriginalURL, "https://go.dev")
	}
}

func TestHandleStats_NotFound(t *testing.T) {
	svc := newTestService(newMockStorage())
	handler := handleStats(svc)

	req := httptest.NewRequest(http.MethodGet, "/stats/nonexistent", nil)
	req.SetPathValue("shortCode", "nonexistent")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
