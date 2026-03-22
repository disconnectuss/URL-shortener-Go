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

	"url-shortener/internal/cache"
	"url-shortener/internal/config"
	"url-shortener/internal/server"
	"url-shortener/internal/service"
	"url-shortener/internal/storage"
	pb "url-shortener/proto"

	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()

	store, err := newStorage(cfg.DBDriver, cfg.DBDsn)
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer store.Close()
	log.Printf("Connected to database (%s)", cfg.DBDriver)

	var c *cache.Cache
	if cfg.RedisAddr != "" {
		c, err = cache.New(cfg.RedisAddr)
		if err != nil {
			log.Printf("Redis not available, running without cache: %v", err)
		} else {
			defer c.Close()
			log.Printf("Connected to Redis (%s)", cfg.RedisAddr)
		}
	}

	go cleanupLoop(store, 10*time.Minute)

	svc := service.New(store, c, cfg.BaseURL)

	httpHandler := server.NewHTTPHandler(svc, cfg.RateLimit, cfg.RateBurst)
	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: httpHandler,
	}

	go func() {
		fmt.Printf("HTTP server is running on: %s\n", cfg.BaseURL)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("HTTP server error:", err)
		}
	}()

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("gRPC listen failed:", err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterURLShortenerServer(grpcSrv, server.NewGRPCServer(svc))

	go func() {
		fmt.Printf("gRPC server is running on: :%s\n", cfg.GRPCPort)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatal("gRPC server error:", err)
		}
	}()

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

func newStorage(driver, dsn string) (storage.Storage, error) {
	switch driver {
	case "postgres":
		return storage.NewPostgres(dsn)
	default:
		return storage.NewSQLite(dsn)
	}
}

func cleanupLoop(store storage.Storage, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		count, err := store.CleanupExpired()
		if err != nil {
			log.Println("cleanup error:", err)
			continue
		}
		if count > 0 {
			log.Printf("cleaned up %d expired URLs", count)
		}
	}
}
