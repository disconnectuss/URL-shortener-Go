package main

import (
	"context"
	"fmt"
	"log/slog"
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
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg := config.Load()

	store, err := newStorage(cfg.DBDriver, cfg.DBDsn)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	slog.Info("connected to database", "driver", cfg.DBDriver)

	var c *cache.Cache
	if cfg.RedisAddr != "" {
		c, err = cache.New(cfg.RedisAddr)
		if err != nil {
			slog.Warn("redis not available, running without cache", "error", err)
		} else {
			defer c.Close()
			slog.Info("connected to redis", "addr", cfg.RedisAddr)
		}
	}

	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	go cleanupLoop(cleanupCtx, store, 10*time.Minute)

	svc := service.New(store, c, cfg.BaseURL)

	// HTTP Server
	httpHandler := server.NewHTTPHandler(svc, cfg.RateLimit, cfg.RateBurst)
	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: httpHandler,
	}

	go func() {
		slog.Info("http server started", "addr", cfg.BaseURL)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	// gRPC Server
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		slog.Error("grpc listen failed", "error", err)
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterURLShortenerServer(grpcSrv, server.NewGRPCServer(svc))

	go func() {
		slog.Info("grpc server started", "addr", ":"+cfg.GRPCPort)
		if err := grpcSrv.Serve(lis); err != nil {
			slog.Error("grpc server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down servers...")

	cleanupCancel()

	grpcSrv.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("http server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("all servers stopped gracefully")
}

func newStorage(driver, dsn string) (storage.Storage, error) {
	switch driver {
	case "postgres":
		return storage.NewPostgres(dsn)
	default:
		return storage.NewSQLite(dsn)
	}
}

func cleanupLoop(ctx context.Context, store storage.Storage, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := store.CleanupExpired(ctx)
			if err != nil {
				slog.Error("cleanup failed", "error", err)
				continue
			}
			if count > 0 {
				slog.Info("cleaned up expired urls", "count", count)
			}
		}
	}
}
