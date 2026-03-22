package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type URLStore struct {
	mu   sync.RWMutex
	urls map[string]string
}

func NewURLStore() *URLStore {
	return &URLStore{
		urls: make(map[string]string),
	}
}

func (s *URLStore) Save(shortCode, originalURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls[shortCode] = originalURL
}

func (s *URLStore) Get(shortCode string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.urls[shortCode]
	return url, ok
}

func generateShortCode() (string, error) {
	bytes := make([]byte, 3) 
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

func main() {
	store := NewURLStore()
	mux := http.NewServeMux()

	mux.HandleFunc("POST /shorten", func(w http.ResponseWriter, r *http.Request) {
		var req ShortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.URL == "" {
			http.Error(w, "url field is required", http.StatusBadRequest)
			return
		}

		code, err := generateShortCode()
		if err != nil {
			http.Error(w, "short code could not be generated", http.StatusInternalServerError)
			return
		}

		store.Save(code, req.URL)

		resp := ShortenResponse{
			ShortURL: fmt.Sprintf("http://localhost:8080/%s", code),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("GET /{shortCode}", func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("shortCode")

		originalURL, ok := store.Get(code)
		if !ok {
			http.Error(w, "URL bulunamadı", http.StatusNotFound)
			return
		}

		http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
	})

	fmt.Println("Server is running on: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
