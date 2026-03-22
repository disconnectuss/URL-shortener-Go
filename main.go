package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

type URLStore struct {
	db *sql.DB
}

func NewURLStore(dbPath string) (*URLStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			short_code TEXT UNIQUE NOT NULL,
			original_url TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, err
	}

	return &URLStore{db: db}, nil
}

func (s *URLStore) Save(shortCode, originalURL string) error {
	_, err := s.db.Exec(
		"INSERT INTO urls (short_code, original_url) VALUES (?, ?)",
		shortCode, originalURL,
	)
	return err
}

func (s *URLStore) Get(shortCode string) (string, error) {
	var originalURL string
	err := s.db.QueryRow(
		"SELECT original_url FROM urls WHERE short_code = ?",
		shortCode,
	).Scan(&originalURL)
	return originalURL, err
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
	store, err := NewURLStore("urls.db")
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer store.db.Close()

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
	})

	mux.HandleFunc("GET /{shortCode}", func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("shortCode")

		originalURL, err := store.Get(code)
		if err != nil {
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}

		http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
	})

	fmt.Println("Server is running on: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
