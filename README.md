# BMKG Earthquake API

![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-green)

API layanan data gempabumi Indonesia yang mengambil data real-time dari [BMKG (Badan Meteorologi, Klimatologi, dan Geofisika)](https://data.bmkg.go.id/gempabumi/).

> **⚠️ Atribusi**: Wajib untuk mencantumkan BMKG sebagai sumber data dan menampilkannya pada aplikasi/sistem Anda sesuai ketentuan yang berlaku.

## Fitur

- 📡 **Data Real-time** — Mengambil data gempabumi terbaru langsung dari BMKG
- 🔄 **Tiga Jenis Data** — Gempa terbaru, M 5.0+, dan gempa dirasakan
- 🖼️ **Shakemap** — Proxy/redirect ke peta guncangan (shakemap)
- ⚡ **Caching** — In-memory cache untuk mengurangi permintaan ke BMKG (rate limit: 60 req/menit/IP)
- 🔒 **Production Ready** — Graceful shutdown, timeout, retry, CORS, panic recovery
- 🐳 **Docker Support** — Dockerfile & Docker Compose siap pakai

## Arsitektur

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

Struktur proyek mengikuti **Clean Architecture**:

```
earthquake-api/
├── cmd/
│   └── api/
│       └── main.go              # Entry point
├── internal/
│   ├── bmkg/
│   │   └── client.go            # HTTP client untuk BMKG API
│   ├── cache/
│   │   └── cache.go             # In-memory cache
│   ├── config/
│   │   └── config.go            # Konfigurasi dari env vars
│   ├── handler/
│   │   └── earthquake.go        # HTTP handlers
│   ├── middleware/
│   │   ├── cors.go              # CORS middleware
│   │   ├── logger.go            # Request logging
│   │   ├── recovery.go          # Panic recovery
│   │   └── requestid.go         # Request ID tracing
│   ├── model/
│   │   ├── earthquake.go        # Struct data gempa
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

- Go 1.22+ atau Docker

### Menjalankan dengan Go

```bash
# Clone repositori
git clone <repo-url>
cd earthquake-api

# Salin konfigurasi
cp .env.example .env

# Jalankan
make run
```

### Menjalankan dengan Docker

```bash
# Build & run dengan Docker Compose
make docker-compose-up

# Atau manual
make docker-build
make docker-run
```

### Hot Reload (Development)

```bash
make run-hot
```

## API Endpoints

### 1. Informasi API

```
GET /
```

Menampilkan informasi API dan daftar endpoint yang tersedia.

### 2. Health Check

```
GET /health
```

Cek kesehatan service.

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

### 3. Gempa Terbaru

```
GET /api/v1/earthquake/latest
```

Mengambil data gempabumi terkini dari BMKG.

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

### 4. Daftar 15 Gempa M 5.0+

```
GET /api/v1/earthquake/list/magnitude5
```

Mengambil 15 gempabumi terakhir dengan magnitudo ≥ 5.0.

### 5. Daftar 15 Gempa Dirasakan

```
GET /api/v1/earthquake/list/felt
```

Mengambil 15 gempabumi terakhir yang dirasakan.

### 6. Shakemap

```
GET /api/v1/earthquake/shakemap?code=20260619085320.mmi.jpg
```

Redirect ke gambar shakemap di server statis BMKG.

## Format Respons

Semua respons menggunakan format JSON envelope yang konsisten:

### Sukses

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

### Kode Error

| HTTP Status | Code | Deskripsi |
|-------------|------|-----------|
| 400 | `INVALID_REQUEST` | Parameter tidak valid |
| 404 | `NOT_FOUND` | Data tidak ditemukan |
| 429 | `UPSTREAM_RATE_LIMITED` | Rate limit BMKG tercapai |
| 503 | `UPSTREAM_ERROR` | Gagal mengambil data dari BMKG |

## Konfigurasi Lingkungan

Semua konfigurasi melalui environment variables (lihat [.env.example](.env.example)):

| Variable | Default | Deskripsi |
|----------|---------|-----------|
| `SERVER_HOST` | `0.0.0.0` | Host server |
| `SERVER_PORT` | `8080` | Port server |
| `SERVER_READ_TIMEOUT` | `10s` | Read timeout |
| `SERVER_WRITE_TIMEOUT` | `30s` | Write timeout |
| `SERVER_IDLE_TIMEOUT` | `60s` | Idle timeout |
| `BMKG_BASE_URL` | `https://data.bmkg.go.id` | Base URL BMKG |
| `BMKG_HTTP_TIMEOUT` | `10s` | HTTP client timeout |
| `BMKG_MAX_RETRIES` | `2` | Jumlah retry ke BMKG |
| `BMKG_RETRY_WAIT` | `500ms` | Interval antar retry |
| `CACHE_ENABLED` | `true` | Aktifkan caching |
| `CACHE_TTL` | `30s` | Cache TTL |
| `CORS_ALLOWED_ORIGINS` | `*` | Origin yang diizinkan |

## Rate Limiting

BMKG menerapkan batas akses **60 permintaan per menit per IP**. API ini menggunakan in-memory cache (default 30 detik) untuk mengurangi jumlah permintaan langsung ke BMKG.

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

## Sumber Data

Data diperoleh dari portal resmi BMKG:
- https://data.bmkg.go.id/gempabumi/
- https://github.com/infoBMKG/data-gempabumi

Endpoint yang digunakan:
- `autogempa.json` — Gempa terbaru
- `gempaterkini.json` — 15 gempa M 5.0+ terbaru
- `gempadirasakan.json` — 15 gempa dirasakan terbaru
- `static.bmkg.go.id/[kode].jpg` — Gambar shakemap

## Lisensi

MIT License

## Atribusi

Data gempabumi berasal dari **BMKG (Badan Meteorologi, Klimatologi, dan Geofisika)**. Wajib mencantumkan BMKG sebagai sumber data sesuai dengan ketentuan yang berlaku.
