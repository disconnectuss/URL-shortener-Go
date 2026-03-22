package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type ShortenRequest struct {
	URL       string `json:"url"`
	ExpiresIn string `json:"expires_in,omitempty"` // ex: "30m", "24h", "7d"
}

type ShortenResponse struct {
	ShortURL  string  `json:"short_url"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

func handleHome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	}
}

func handleShorten(store *URLStore, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ShortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.URL == "" {
			http.Error(w, "url field is required", http.StatusBadRequest)
			return
		}

		parsedURL, err := url.ParseRequestURI(req.URL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			http.Error(w, "invalid URL: must start with http:// or https://", http.StatusBadRequest)
			return
		}
		if parsedURL.Host == "" {
			http.Error(w, "invalid URL: missing host", http.StatusBadRequest)
			return
		}

		var expiresAt *time.Time
		if req.ExpiresIn != "" {
			d, err := parseDuration(req.ExpiresIn)
			if err != nil {
				http.Error(w, "invalid expires_in: use format like 30m, 24h, 7d", http.StatusBadRequest)
				return
			}
			t := time.Now().Add(d)
			expiresAt = &t
		}

		code, err := generateShortCode()
		if err != nil {
			http.Error(w, "short code could not be generated", http.StatusInternalServerError)
			return
		}

		if err := store.Save(code, req.URL, expiresAt); err != nil {
			http.Error(w, "could not save URL", http.StatusInternalServerError)
			return
		}

		resp := ShortenResponse{
			ShortURL: fmt.Sprintf("%s/%s", baseURL, code),
		}
		if expiresAt != nil {
			formatted := expiresAt.UTC().Format(time.RFC3339)
			resp.ExpiresAt = &formatted
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func handleRedirect(store *URLStore, cache *Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("shortCode")

		// Önce cache'e bak
		if cache != nil {
			if url, err := cache.Get(code); err == nil {
				store.IncrementClick(code)
				http.Redirect(w, r, url, http.StatusMovedPermanently)
				return
			}
		}

		// Cache'de yoksa DB'den oku
		originalURL, err := store.Get(code)
		if err != nil {
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}

		// Cache'e yaz
		if cache != nil {
			cache.Set(code, originalURL)
		}

		store.IncrementClick(code)
		http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
	}
}

func handleStats(store *URLStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("shortCode")

		stats, err := store.GetStats(code)
		if err != nil {
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
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
		// "7d" → "168h" convertion
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
