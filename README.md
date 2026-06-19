# BMKG Earthquake API

![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-green)

A REST API service for Indonesian earthquake data, fetching real-time information from [BMKG (Meteorological, Climatological, and Geophysical Agency)](https://data.bmkg.go.id/gempabumi/).

> **⚠️ Attribution**: You must credit BMKG as the data source and display it in your application/system according to applicable regulations.

## Features

- 📡 **Real-time Data** — Fetches the latest earthquake data directly from BMKG
- 🔄 **Three Data Types** — Latest earthquake, M 5.0+, and felt earthquakes
- 🖼️ **Shakemap** — Proxy/redirect to shake maps (shakemap images)
- ⚡ **Caching** — In-memory cache to reduce requests to BMKG (rate limit: 60 req/min/IP)
- 🔒 **Production Ready** — Graceful shutdown, timeout, retry, CORS, panic recovery
- 🐳 **Docker Support** — Dockerfile & Docker Compose ready to use

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐     ┌──────────────────────┐
│   Client    │────▶│  HTTP Server  │────▶│  Service    │────▶│  BMKG Client         │
│ (Browser/App)│    │  (net/http)   │     │  + Cache    │     │  (data.bmkg.go.id)   │
└─────────────┘     └──────────────┘     └─────────────┘     └──────────────────────┘
                          │
                    ┌─────┴──────┐
                    │ Middleware  │
                    │ • Recovery  │
                    │ • CORS      │
                    │ • Logger    │
                    │ • RequestID │
                    └────────────┘
```

The project structure follows **Clean Architecture**:

```
earthquake-api/
├── cmd/
│   └── api/
│       └── main.go              # Entry point
├── internal/
│   ├── bmkg/
│   │   └── client.go            # HTTP client for BMKG API
│   ├── cache/
│   │   └── cache.go             # In-memory cache
│   ├── config/
│   │   └── config.go            # Configuration from env vars
│   ├── handler/
│   │   └── earthquake.go        # HTTP handlers
│   ├── middleware/
│   │   ├── cors.go              # CORS middleware
│   │   ├── logger.go            # Request logging
│   │   ├── recovery.go          # Panic recovery
│   │   └── requestid.go         # Request ID tracing
│   ├── model/
│   │   ├── earthquake.go        # Earthquake data structs
│   │   └── errors.go            # Error definitions
│   ├── response/
│   │   └── response.go          # Standard JSON envelope
│   └── service/
│       └── earthquake.go        # Business logic + caching
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── .env.example
└── README.md
```

## Quick Start

### Prerequisites

- Go 1.22+ or Docker

### Running with Go

```bash
# Clone the repository
git clone https://github.com/ibnuali/bmkg-earthquake-api
cd earthquake-api

# Copy configuration
cp .env.example .env

# Run
make run
```

### Running with Docker

```bash
# Build & run with Docker Compose
make docker-compose-up

# Or manually
make docker-build
make docker-run
```

### Hot Reload (Development)

```bash
make run-hot
```

## API Endpoints

### 1. API Information

```
GET /
```

Displays API information and a list of available endpoints.

### 2. Health Check

```
GET /health
```

Check service health.

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "healthy"
  },
  "error": null,
  "metadata": {
    "source": "BMKG",
    "api_version": "v1",
    "timestamp": "2026-06-19T01:53:20Z"
  }
}
```

### 3. Latest Earthquake

```
GET /api/v1/earthquake/latest
```

Fetches the most recent earthquake data from BMKG.

**Response:**
```json
{
  "success": true,
  "data": {
    "date": "19 Jun 2026",
    "time": "08:53:20 WIB",
    "datetime": "2026-06-19T01:53:20Z",
    "coordinates": [-1.17, 120.01],
    "latitude": -1.17,
    "longitude": 120.01,
    "magnitude": 3.3,
    "depth_km": 4,
    "region": "Pusat gempa berada di darat 28 km timur laut Sigi",
    "potency": "Gempa ini dirasakan untuk diteruskan pada masyarakat",
    "felt": "",
    "shakemap_url": "https://static.bmkg.go.id/20260619085320.mmi.jpg"
  },
  "error": null,
  "metadata": {
    "source": "BMKG",
    "api_version": "v1",
    "timestamp": "2026-06-19T02:00:00Z"
  }
}
```

### 4. List of 15 Earthquakes M 5.0+

```
GET /api/v1/earthquake/list/magnitude5
```

Fetches the 15 most recent earthquakes with magnitude ≥ 5.0.

### 5. List of 15 Felt Earthquakes

```
GET /api/v1/earthquake/list/felt
```

Fetches the 15 most recent felt earthquakes.

### 6. Shakemap

```
GET /api/v1/earthquake/shakemap?code=20260619085320.mmi.jpg
```

Redirects to the shakemap image on BMKG's static server.

## Response Format

All responses use a consistent JSON envelope format:

### Success

```json
{
  "success": true,
  "data": { ... },
  "error": null,
  "metadata": {
    "source": "BMKG",
    "api_version": "v1",
    "timestamp": "2026-06-19T01:53:20Z"
  }
}
```

### Error

```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "UPSTREAM_ERROR",
    "message": "Failed to fetch data from BMKG"
  },
  "metadata": {
    "source": "BMKG",
    "api_version": "v1",
    "timestamp": "2026-06-19T01:53:20Z"
  }
}
```

### Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | `INVALID_REQUEST` | Invalid parameters |
| 404 | `NOT_FOUND` | Data not found |
| 429 | `UPSTREAM_RATE_LIMITED` | BMKG rate limit reached |
| 503 | `UPSTREAM_ERROR` | Failed to fetch data from BMKG |

## Environment Configuration

All configuration is done through environment variables (see [.env.example](.env.example)):

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | `0.0.0.0` | Server host |
| `SERVER_PORT` | `8080` | Server port |
| `SERVER_READ_TIMEOUT` | `10s` | Read timeout |
| `SERVER_WRITE_TIMEOUT` | `30s` | Write timeout |
| `SERVER_IDLE_TIMEOUT` | `60s` | Idle timeout |
| `BMKG_BASE_URL` | `https://data.bmkg.go.id` | BMKG base URL |
| `BMKG_HTTP_TIMEOUT` | `10s` | HTTP client timeout |
| `BMKG_MAX_RETRIES` | `2` | Number of retries to BMKG |
| `BMKG_RETRY_WAIT` | `500ms` | Interval between retries |
| `CACHE_ENABLED` | `true` | Enable caching |
| `CACHE_TTL` | `30s` | Cache TTL |
| `CORS_ALLOWED_ORIGINS` | `*` | Allowed origins |

## Rate Limiting

BMKG enforces a limit of **60 requests per minute per IP**. This API uses in-memory caching (default 30 seconds) to reduce the number of direct requests to BMKG.

## Development

### Testing

```bash
make test
```

### Linting

```bash
make lint
```

### Build

```bash
make build
```

## Data Source

Data is obtained from the official BMKG portal:
- https://data.bmkg.go.id/gempabumi/
- https://github.com/infoBMKG/data-gempabumi

Endpoints used:
- `autogempa.json` — Latest earthquake
- `gempaterkini.json` — 15 most recent M 5.0+ earthquakes
- `gempadirasakan.json` — 15 most recent felt earthquakes
- `static.bmkg.go.id/[code].jpg` — Shakemap images

## License

MIT License

## Attribution

Earthquake data is sourced from **BMKG (Badan Meteorologi, Klimatologi, dan Geofisika)**. You must credit BMKG as the data source in accordance with applicable regulations.
