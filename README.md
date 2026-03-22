# URL Shortener

A fast, lightweight URL shortener built with Go. Supports both HTTP and gRPC APIs, optional Redis caching, rate limiting, and automatic URL expiration.

## Features

- **Dual API** — REST (HTTP) and gRPC interfaces
- **Two database backends** — SQLite (default, zero config) or PostgreSQL
- **Redis caching** — Optional, with graceful degradation when unavailable
- **URL expiration** — Set TTLs like `30m`, `24h`, `7d`; expired URLs are cleaned up automatically
- **Rate limiting** — Per-IP token bucket with configurable rate and burst
- **Web UI** — Clean dark-themed frontend at `/`

## Quick Start

```bash
# Run with SQLite (no dependencies needed)
go run ./cmd/server/
```

The server starts at http://localhost:8080. Open it in your browser to use the web UI.

### With PostgreSQL and Redis

```bash
# Start PostgreSQL and Redis
docker-compose up -d

# Run the server with Postgres + Redis
DB_DRIVER=postgres REDIS_ADDR=localhost:6379 go run ./cmd/server/
```

## API

### HTTP Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET /` | Serves the web UI |
| `POST /shorten` | Create a short URL |
| `GET /{code}` | Redirect to original URL |
| `GET /stats/{code}` | Get click stats for a URL |

**Shorten a URL:**

```bash
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "expires_in": "24h"}'
```

```json
{
  "short_url": "http://localhost:8080/a1b2c3d4",
  "expires_at": "2026-03-23T22:00:00Z"
}
```

**Get stats:**

```bash
curl http://localhost:8080/stats/a1b2c3d4
```

```json
{
  "short_code": "a1b2c3d4",
  "original_url": "https://example.com",
  "click_count": 5,
  "created_at": "2026-03-22T22:00:00Z",
  "expires_at": "2026-03-23T22:00:00Z"
}
```

### gRPC

The gRPC server runs on port `9090` by default. See [`proto/urlshortener.proto`](proto/urlshortener.proto) for the service definition.

```protobuf
service URLShortener {
  rpc Shorten(ShortenRequest)  returns (ShortenResponse);
  rpc Resolve(ResolveRequest)  returns (ResolveResponse);
  rpc GetStats(StatsRequest)   returns (StatsResponse);
}
```

A test client is included:

```bash
go run ./cmd/grpc-test/
```

## Configuration

All settings are configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `GRPC_PORT` | `9090` | gRPC server port |
| `DB_DRIVER` | `sqlite3` | Database driver (`sqlite3` or `postgres`) |
| `DB_PATH` | `urls.db` | SQLite file path |
| `DATABASE_URL` | `postgres://urlshortener:urlshortener@localhost:5433/urlshortener?sslmode=disable` | PostgreSQL connection string |
| `BASE_URL` | `http://localhost:{PORT}` | Base URL for generated short links |
| `REDIS_ADDR` | _(empty)_ | Redis address (e.g. `localhost:6379`). Empty = no caching |
| `RATE_LIMIT` | `10` | Max requests per second per IP |
| `RATE_BURST` | `20` | Burst capacity per IP |

## Project Structure

```
├── cmd/
│   ├── server/          # Application entry point
│   └── grpc-test/       # gRPC test client
├── internal/
│   ├── cache/           # Redis caching layer
│   ├── config/          # Environment-based configuration
│   ├── model/           # Request/response models
│   ├── server/          # HTTP & gRPC handlers, middleware
│   ├── service/         # Core business logic
│   └── storage/         # Database layer (SQLite, PostgreSQL)
├── proto/               # Protobuf definitions & generated code
├── templates/           # Web UI
└── docker-compose.yml   # PostgreSQL & Redis services
```

## Testing

```bash
go test ./...
```

## Tech Stack

- **Go 1.26** — Language
- **SQLite / PostgreSQL** — Storage
- **Redis** — Caching
- **gRPC + Protocol Buffers** — RPC interface
- **Docker Compose** — Local infrastructure
