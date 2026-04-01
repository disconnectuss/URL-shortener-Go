package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"url-shortener/internal/model"
	"url-shortener/internal/service"
)

func NewHTTPHandler(svc *service.URLService, rateLimit, rateBurst int) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", handleHome())
	mux.HandleFunc("POST /shorten", handleShorten(svc))
	mux.HandleFunc("GET /stats/{shortCode}", handleStats(svc))
	mux.HandleFunc("GET /{shortCode}", handleRedirect(svc))

	rl := newRateLimiter(rateLimit, rateBurst)
	return loggingMiddleware(rateLimitMiddleware(rl)(mux))
}

func handleHome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	}
}

func handleShorten(svc *service.URLService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.ShortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		resp, err := svc.Shorten(r.Context(), req.URL, req.ExpiresIn)
		if err != nil {
			http.Error(w, err.Error(), errorToStatus(err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func handleRedirect(svc *service.URLService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("shortCode")

		originalURL, err := svc.Resolve(r.Context(), code)
		if err != nil {
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}

		http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
	}
}

func handleStats(svc *service.URLService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("shortCode")

		stats, err := svc.GetStats(r.Context(), code)
		if err != nil {
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

func errorToStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
