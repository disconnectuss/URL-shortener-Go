package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func setupTestServer(t *testing.T) (*URLStore, *http.ServeMux) {
	t.Helper()
	dbPath := "test_" + t.Name() + ".db"
	t.Cleanup(func() { os.Remove(dbPath) })

	store, err := NewURLStore("sqlite3", dbPath)
	if err != nil {
		t.Fatal("failed to create store:", err)
	}
	t.Cleanup(func() { store.Close() })

	mux := http.NewServeMux()
	mux.HandleFunc("POST /shorten", handleShorten(store, "http://localhost:8080"))
	mux.HandleFunc("GET /stats/{shortCode}", handleStats(store))
	mux.HandleFunc("GET /{shortCode}", handleRedirect(store, nil))

	return store, mux
}

func TestHandleShorten(t *testing.T) {
	_, mux := setupTestServer(t)

	body := `{"url": "https://go.dev"}`
	req := httptest.NewRequest("POST", "/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp ShortenResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ShortURL == "" {
		t.Error("short_url should not be empty")
	}
}

func TestHandleShortenInvalidURL(t *testing.T) {
	_, mux := setupTestServer(t)

	tests := []struct {
		name string
		body string
	}{
		{"empty url", `{"url": ""}`},
		{"no scheme", `{"url": "go.dev"}`},
		{"ftp scheme", `{"url": "ftp://example.com"}`},
		{"invalid json", `not json`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/shorten", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleRedirect(t *testing.T) {
	store, mux := setupTestServer(t)

	store.Save("redir1", "https://go.dev", nil)

	req := httptest.NewRequest("GET", "/redir1", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMovedPermanently)
	}
	if loc := w.Header().Get("Location"); loc != "https://go.dev" {
		t.Errorf("Location = %q, want %q", loc, "https://go.dev")
	}
}

func TestHandleRedirectNotFound(t *testing.T) {
	_, mux := setupTestServer(t)

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleStats(t *testing.T) {
	store, mux := setupTestServer(t)

	store.Save("stat1", "https://go.dev", nil)
	store.IncrementClick("stat1")
	store.IncrementClick("stat1")

	req := httptest.NewRequest("GET", "/stats/stat1", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var stats URLStats
	json.NewDecoder(w.Body).Decode(&stats)
	if stats.ClickCount != 2 {
		t.Errorf("click_count = %d, want 2", stats.ClickCount)
	}
}

func TestRedirectIncrementsClickCount(t *testing.T) {
	store, mux := setupTestServer(t)

	store.Save("cnt01", "https://go.dev", nil)

	for range 3 {
		req := httptest.NewRequest("GET", "/cnt01", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}

	stats, _ := store.GetStats("cnt01")
	if stats.ClickCount != 3 {
		t.Errorf("click_count = %d, want 3", stats.ClickCount)
	}
}

func TestHandleShortenWithExpiry(t *testing.T) {
	_, mux := setupTestServer(t)

	body := `{"url": "https://go.dev", "expires_in": "24h"}`
	req := httptest.NewRequest("POST", "/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp ShortenResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ExpiresAt == nil {
		t.Error("expires_at should not be nil when expires_in is set")
	}
}

func TestHandleShortenInvalidExpiry(t *testing.T) {
	_, mux := setupTestServer(t)

	body := `{"url": "https://go.dev", "expires_in": "abc"}`
	req := httptest.NewRequest("POST", "/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
