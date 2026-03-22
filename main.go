package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	store, err := NewURLStore("urls.db")
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer store.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /shorten", handleShorten(store))
	mux.HandleFunc("GET /stats/{shortCode}", handleStats(store))
	mux.HandleFunc("GET /{shortCode}", handleRedirect(store))

	handler := loggingMiddleware(mux)

	fmt.Println("Server is running on: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
