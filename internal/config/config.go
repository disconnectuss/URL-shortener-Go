package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port      string
	GRPCPort  string
	DBDriver  string
	DBDsn     string
	RateLimit int
	RateBurst int
	BaseURL   string
	RedisAddr string
}

func Load() *Config {
	driver := getEnv("DB_DRIVER", "sqlite3")
	var dsn string
	if driver == "postgres" {
		dsn = getEnv("DATABASE_URL", "postgres://urlshortener:urlshortener@localhost:5433/urlshortener?sslmode=disable")
	} else {
		dsn = getEnv("DB_PATH", "urls.db")
	}

	cfg := &Config{
		Port:      getEnv("PORT", "8080"),
		GRPCPort:  getEnv("GRPC_PORT", "9090"),
		DBDriver:  driver,
		DBDsn:     dsn,
		RateLimit: getEnvInt("RATE_LIMIT", 10),
		RateBurst: getEnvInt("RATE_BURST", 20),
		RedisAddr: getEnv("REDIS_ADDR", ""),
	}
	cfg.BaseURL = getEnv("BASE_URL", fmt.Sprintf("http://localhost:%s", cfg.Port))
	return cfg
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}
