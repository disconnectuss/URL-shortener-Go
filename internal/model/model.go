package model

import "time"

type URL struct {
	ID          int
	ShortCode   string
	OriginalURL string
	ClickCount  int
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}

type URLStats struct {
	ShortCode   string  `json:"short_code"`
	OriginalURL string  `json:"original_url"`
	ClickCount  int     `json:"click_count"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
}

type ShortenRequest struct {
	URL       string `json:"url"`
	ExpiresIn string `json:"expires_in,omitempty"`
}

type ShortenResponse struct {
	ShortURL  string  `json:"short_url"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}
