package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig
	BMKG     BMKGConfig
	Cache    CacheConfig
	CORS     CORSConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// BMKGConfig holds BMKG API client configuration.
type BMKGConfig struct {
	BaseURL       string
	JSONEndpoint  string
	ShakemapURL   string
	HTTPTimeout   time.Duration
	MaxRetries    int
	RetryWait     time.Duration
}

// CacheConfig holds in-memory cache configuration.
type CacheConfig struct {
	Enabled     bool
	TTL         time.Duration
	CleanupIntv time.Duration
}

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	// Load .env file if it exists (won't override already-set env vars)
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnv("SERVER_PORT", "8090"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		BMKG: BMKGConfig{
			BaseURL:      getEnv("BMKG_BASE_URL", "https://data.bmkg.go.id"),
			JSONEndpoint: getEnv("BMKG_JSON_ENDPOINT", "/DataMKG/TEWS"),
			ShakemapURL:  getEnv("BMKG_SHAKEMAP_URL", "https://static.bmkg.go.id"),
			HTTPTimeout:  getDurationEnv("BMKG_HTTP_TIMEOUT", 10*time.Second),
			MaxRetries:   getIntEnv("BMKG_MAX_RETRIES", 2),
			RetryWait:    getDurationEnv("BMKG_RETRY_WAIT", 500*time.Millisecond),
		},
		Cache: CacheConfig{
			Enabled:     getBoolEnv("CACHE_ENABLED", true),
			TTL:         getDurationEnv("CACHE_TTL", 30*time.Second),
			CleanupIntv: getDurationEnv("CACHE_CLEANUP_INTERVAL", 60*time.Second),
		},
		CORS: CORSConfig{
			AllowedOrigins: getStringSliceEnv("CORS_ALLOWED_ORIGINS", []string{"*"}),
			AllowedMethods: []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders: []string{"Accept", "Content-Type", "X-Request-ID"},
		},
	}
}

// ListenAddr returns the address string the server should listen on.
func (s ServerConfig) ListenAddr() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getStringSliceEnv(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		return []string{v}
	}
	return fallback
}
