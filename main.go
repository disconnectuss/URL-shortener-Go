package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "url-shortener/proto"

	"google.golang.org/grpc"
)

func main() {
	cfg := LoadConfig()

	store, err := NewURLStore(cfg.DBDriver, cfg.DBDsn)
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer store.Close()
	log.Printf("Connected to database (%s)", cfg.DBDriver)

	var cache *Cache
	if cfg.RedisAddr != "" {
		cache, err = NewCache(cfg.RedisAddr)
		if err != nil {
			log.Printf("Redis not available, running without cache: %v", err)
		} else {
			defer cache.Close()
			log.Printf("Connected to Redis (%s)", cfg.RedisAddr)
		}
	}

	go store.CleanupExpired(10 * time.Minute)

	// --- HTTP Server ---
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleHome())
	mux.HandleFunc("POST /shorten", handleShorten(store, cfg.BaseURL))
	mux.HandleFunc("GET /stats/{shortCode}", handleStats(store))
	mux.HandleFunc("GET /{shortCode}", handleRedirect(store, cache))

	rl := newRateLimiter(cfg.RateLimit, cfg.RateBurst)
	handler := loggingMiddleware(rateLimitMiddleware(rl)(mux))

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}

	go func() {
		fmt.Printf("HTTP server is running on: %s\n", cfg.BaseURL)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("HTTP server error:", err)
		}
	}()

	// --- gRPC Server ---
	grpcPort := getEnv("GRPC_PORT", "9090")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatal("gRPC listen failed:", err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterURLShortenerServer(grpcSrv, newGRPCServer(store, cfg.BaseURL))

	go func() {
		fmt.Printf("gRPC server is running on: :%s\n", grpcPort)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatal("gRPC server error:", err)
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down servers...")

	grpcSrv.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal("HTTP server forced to shutdown:", err)
	}

	fmt.Println("All servers stopped gracefully")
}
