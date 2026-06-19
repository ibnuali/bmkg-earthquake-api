package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	unsetEnvVars(t)

	cfg := Load()

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected host '0.0.0.0', got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != "8080" {
		t.Errorf("expected port '8080', got %s", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 10*time.Second {
		t.Errorf("expected read timeout 10s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 30*time.Second {
		t.Errorf("expected write timeout 30s, got %v", cfg.Server.WriteTimeout)
	}
	if cfg.Server.IdleTimeout != 60*time.Second {
		t.Errorf("expected idle timeout 60s, got %v", cfg.Server.IdleTimeout)
	}

	if cfg.BMKG.BaseURL != "https://data.bmkg.go.id" {
		t.Errorf("expected BMKG base URL, got %s", cfg.BMKG.BaseURL)
	}
	if cfg.BMKG.JSONEndpoint != "/DataMKG/TEWS" {
		t.Errorf("expected JSON endpoint, got %s", cfg.BMKG.JSONEndpoint)
	}
	if cfg.BMKG.ShakemapURL != "https://static.bmkg.go.id" {
		t.Errorf("expected shakemap URL, got %s", cfg.BMKG.ShakemapURL)
	}
	if cfg.BMKG.HTTPTimeout != 10*time.Second {
		t.Errorf("expected HTTP timeout 10s, got %v", cfg.BMKG.HTTPTimeout)
	}
	if cfg.BMKG.MaxRetries != 2 {
		t.Errorf("expected max retries 2, got %d", cfg.BMKG.MaxRetries)
	}
	if cfg.BMKG.RetryWait != 500*time.Millisecond {
		t.Errorf("expected retry wait 500ms, got %v", cfg.BMKG.RetryWait)
	}

	if !cfg.Cache.Enabled {
		t.Error("expected cache enabled=true")
	}
	if cfg.Cache.TTL != 30*time.Second {
		t.Errorf("expected cache TTL 30s, got %v", cfg.Cache.TTL)
	}
	if cfg.Cache.CleanupIntv != 60*time.Second {
		t.Errorf("expected cache cleanup interval 60s, got %v", cfg.Cache.CleanupIntv)
	}

	if len(cfg.CORS.AllowedOrigins) != 1 || cfg.CORS.AllowedOrigins[0] != "*" {
		t.Errorf("expected CORS origins ['*'], got %v", cfg.CORS.AllowedOrigins)
	}
}

func TestLoadWithEnvOverrides(t *testing.T) {
	unsetEnvVars(t)

	os.Setenv("SERVER_HOST", "127.0.0.1")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("SERVER_READ_TIMEOUT", "5s")
	os.Setenv("CACHE_ENABLED", "false")
	os.Setenv("CACHE_TTL", "10s")
	os.Setenv("BMKG_MAX_RETRIES", "5")
	os.Setenv("BMKG_RETRY_WAIT", "1s")
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://example.com")
	defer unsetEnvVars(t)

	cfg := Load()

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("expected host '127.0.0.1', got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != "9090" {
		t.Errorf("expected port '9090', got %s", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 5*time.Second {
		t.Errorf("expected read timeout 5s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Cache.Enabled {
		t.Error("expected cache enabled=false")
	}
	if cfg.Cache.TTL != 10*time.Second {
		t.Errorf("expected cache TTL 10s, got %v", cfg.Cache.TTL)
	}
	if cfg.BMKG.MaxRetries != 5 {
		t.Errorf("expected max retries 5, got %d", cfg.BMKG.MaxRetries)
	}
	if cfg.BMKG.RetryWait != 1*time.Second {
		t.Errorf("expected retry wait 1s, got %v", cfg.BMKG.RetryWait)
	}
	if cfg.CORS.AllowedOrigins[0] != "http://example.com" {
		t.Errorf("expected CORS origin 'http://example.com', got %s", cfg.CORS.AllowedOrigins[0])
	}
}

func TestLoadWithInvalidEnvValues(t *testing.T) {
	unsetEnvVars(t)

	os.Setenv("SERVER_READ_TIMEOUT", "not-a-duration")
	os.Setenv("BMKG_MAX_RETRIES", "not-a-number")
	os.Setenv("CACHE_ENABLED", "not-a-bool")
	defer unsetEnvVars(t)

	cfg := Load()

	if cfg.Server.ReadTimeout != 10*time.Second {
		t.Errorf("expected fallback read timeout 10s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.BMKG.MaxRetries != 2 {
		t.Errorf("expected fallback max retries 2, got %d", cfg.BMKG.MaxRetries)
	}
	if !cfg.Cache.Enabled {
		t.Error("expected fallback cache enabled=true")
	}
}

func TestListenAddr(t *testing.T) {
	tests := []struct {
		host string
		port string
		want string
	}{
		{"0.0.0.0", "8080", "0.0.0.0:8080"},
		{"127.0.0.1", "9090", "127.0.0.1:9090"},
		{"localhost", "3000", "localhost:3000"},
		{"", "", ":"},
	}

	for _, tt := range tests {
		s := ServerConfig{Host: tt.host, Port: tt.port}
		got := s.ListenAddr()
		if got != tt.want {
			t.Errorf("ListenAddr(%q, %q) = %q, want %q", tt.host, tt.port, got, tt.want)
		}
	}
}

func TestGetEnv(t *testing.T) {
	if got := getEnv("NONEXISTENT_VAR_ABC", "fallback"); got != "fallback" {
		t.Errorf("expected 'fallback', got %q", got)
	}

	os.Setenv("EXISTING_VAR", "value")
	defer os.Unsetenv("EXISTING_VAR")

	if got := getEnv("EXISTING_VAR", "fallback"); got != "value" {
		t.Errorf("expected 'value', got %q", got)
	}
}

func TestGetIntEnv(t *testing.T) {
	if got := getIntEnv("NONEXISTENT_INT", 42); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}

	os.Setenv("EXISTING_INT", "99")
	defer os.Unsetenv("EXISTING_INT")

	if got := getIntEnv("EXISTING_INT", 42); got != 99 {
		t.Errorf("expected 99, got %d", got)
	}
}

func TestGetDurationEnv(t *testing.T) {
	if got := getDurationEnv("NONEXISTENT_DUR", time.Minute); got != time.Minute {
		t.Errorf("expected 1m, got %v", got)
	}

	os.Setenv("EXISTING_DUR", "5s")
	defer os.Unsetenv("EXISTING_DUR")

	if got := getDurationEnv("EXISTING_DUR", time.Minute); got != 5*time.Second {
		t.Errorf("expected 5s, got %v", got)
	}
}

func TestGetBoolEnv(t *testing.T) {
	if got := getBoolEnv("NONEXISTENT_BOOL", true); got != true {
		t.Errorf("expected true, got %v", got)
	}

	os.Setenv("EXISTING_BOOL", "false")
	defer os.Unsetenv("EXISTING_BOOL")

	if got := getBoolEnv("EXISTING_BOOL", true); got != false {
		t.Errorf("expected false, got %v", got)
	}
}

func TestGetStringSliceEnv(t *testing.T) {
	if got := getStringSliceEnv("NONEXISTENT_SLICE", []string{"*"}); got[0] != "*" {
		t.Errorf("expected ['*'], got %v", got)
	}

	os.Setenv("EXISTING_SLICE", "http://localhost:3000")
	defer os.Unsetenv("EXISTING_SLICE")

	got := getStringSliceEnv("EXISTING_SLICE", []string{"*"})
	if len(got) != 1 || got[0] != "http://localhost:3000" {
		t.Errorf("expected ['http://localhost:3000'], got %v", got)
	}
}

func unsetEnvVars(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"SERVER_HOST", "SERVER_PORT", "SERVER_READ_TIMEOUT",
		"SERVER_WRITE_TIMEOUT", "SERVER_IDLE_TIMEOUT",
		"BMKG_BASE_URL", "BMKG_JSON_ENDPOINT", "BMKG_SHAKEMAP_URL",
		"BMKG_HTTP_TIMEOUT", "BMKG_MAX_RETRIES", "BMKG_RETRY_WAIT",
		"CACHE_ENABLED", "CACHE_TTL", "CACHE_CLEANUP_INTERVAL",
		"CORS_ALLOWED_ORIGINS",
	} {
		os.Unsetenv(key)
	}
}
