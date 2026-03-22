package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	store, err := NewURLStore("urls.db")
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer store.Close()

	go store.CleanupExpired(10 * time.Minute)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /shorten", handleShorten(store))
	mux.HandleFunc("GET /stats/{shortCode}", handleStats(store))
	mux.HandleFunc("GET /{shortCode}", handleRedirect(store))

	rl := newRateLimiter(10, 20) // 10 requests per minute, burst of 20
	handler := loggingMiddleware(rateLimitMiddleware(rl)(mux))

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	go func() {
		fmt.Println("Server is running on: http://localhost:8080")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("Server error:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	fmt.Println("Server stopped gracefully")
}
