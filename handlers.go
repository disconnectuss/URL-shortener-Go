package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

func handleShorten(store *URLStore) http.HandlerFunc {
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

		code, err := generateShortCode()
		if err != nil {
			http.Error(w, "short code could not be generated", http.StatusInternalServerError)
			return
		}

		if err := store.Save(code, req.URL); err != nil {
			http.Error(w, "could not save URL", http.StatusInternalServerError)
			return
		}

		resp := ShortenResponse{
			ShortURL: fmt.Sprintf("http://localhost:8080/%s", code),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func handleRedirect(store *URLStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("shortCode")

		originalURL, err := store.Get(code)
		if err != nil {
			http.Error(w, "URL not found", http.StatusNotFound)
			return
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
