package bmkg

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"earthquake-api/internal/config"
	"earthquake-api/internal/model"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"valid jpg", "20260619085320.mmi.jpg", false},
		{"valid png", "shakemap.png", false},
		{"valid with slash", "2026/06/19/shakemap.jpg", false},
		{"empty", "", true},
		{"url encoded", "shakemap%2Ejpg", true},
		{"invalid chars", "shakemap<script>.jpg", true},
		{"alphanumeric only", "abc123", false},
		{"dots and hyphens", "file-name.v2.jpg", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.code)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for code %q", tt.code)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for code %q: %v", tt.code, err)
			}
		})
	}
}

// mockServer creates a test HTTP server that responds with the given status and body.
func mockServer(t *testing.T, status int, body interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if body != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(body)
		} else {
			w.WriteHeader(status)
		}
	}))
}

func TestFetchLatest_Success(t *testing.T) {
	resp := model.AutoGempaResponse{}
	resp.Infogempa.Gempa = model.Earthquake{
		Date:        "19 Jun 2026",
		Time:        "08:53:20 WIB",
		DateTime:    "2026-06-19T01:53:20+00:00",
		Coordinates: "-1.17,120.01",
		Magnitude:   "3.3",
		Depth:       "4 km",
		Region:      "Test region",
	}

	srv := mockServer(t, http.StatusOK, resp)
	defer srv.Close()

	cfg := config.BMKGConfig{
		BaseURL:     srv.URL,
		HTTPTimeout: 5 * time.Second,
		MaxRetries:  0,
	}
	client := New(cfg)

	gempa, err := client.FetchLatest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gempa.Magnitude != "3.3" {
		t.Errorf("expected magnitude 3.3, got %s", gempa.Magnitude)
	}
	if gempa.Region != "Test region" {
		t.Errorf("expected region 'Test region', got %s", gempa.Region)
	}
}

func TestFetchLatest_ServerError(t *testing.T) {
	srv := mockServer(t, http.StatusInternalServerError, nil)
	defer srv.Close()

	cfg := config.BMKGConfig{
		BaseURL:     srv.URL,
		HTTPTimeout: 5 * time.Second,
		MaxRetries:  0,
	}
	client := New(cfg)

	_, err := client.FetchLatest()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchLatest_RateLimited(t *testing.T) {
	srv := mockServer(t, http.StatusTooManyRequests, nil)
	defer srv.Close()

	cfg := config.BMKGConfig{
		BaseURL:     srv.URL,
		HTTPTimeout: 5 * time.Second,
		MaxRetries:  1,
	}
	client := New(cfg)

	_, err := client.FetchLatest()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isErrRateLimited(err) {
		t.Errorf("expected rate limit error, got: %v", err)
	}
}

func TestFetchLatest_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer srv.Close()

	cfg := config.BMKGConfig{
		BaseURL:     srv.URL,
		HTTPTimeout: 5 * time.Second,
		MaxRetries:  0,
	}
	client := New(cfg)

	_, err := client.FetchLatest()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestFetchLatest_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	cfg := config.BMKGConfig{
		BaseURL:     srv.URL,
		HTTPTimeout: 50 * time.Millisecond,
		MaxRetries:  0,
	}
	client := New(cfg)

	_, err := client.FetchLatest()
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestFetchLatest_RetryThenSuccess(t *testing.T) {
	attempt := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt <= 1 {
			// First attempt fails
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Second attempt succeeds
		resp := model.AutoGempaResponse{}
		resp.Infogempa.Gempa = model.Earthquake{
			Magnitude: "5.0",
			Depth:     "10 km",
			Region:    "Retry test",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cfg := config.BMKGConfig{
		BaseURL:     srv.URL,
		HTTPTimeout: 5 * time.Second,
		MaxRetries:  1,
		RetryWait:   10 * time.Millisecond,
	}
	client := New(cfg)

	gempa, err := client.FetchLatest()
	if err != nil {
		t.Fatalf("unexpected error after retry: %v", err)
	}
	if gempa.Magnitude != "5.0" {
		t.Errorf("expected magnitude 5.0, got %s", gempa.Magnitude)
	}
	if attempt != 2 {
		t.Errorf("expected 2 attempts, got %d", attempt)
	}
}

func TestFetchLatest_RetryAllFail(t *testing.T) {
	attempt := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := config.BMKGConfig{
		BaseURL:     srv.URL,
		HTTPTimeout: 5 * time.Second,
		MaxRetries:  2,
		RetryWait:   5 * time.Millisecond,
	}
	client := New(cfg)

	_, err := client.FetchLatest()
	if err == nil {
		t.Fatal("expected error after all retries fail")
	}
	if attempt != 3 { // initial + 2 retries
		t.Errorf("expected 3 attempts, got %d", attempt)
	}
}

func TestFetchM5Plus_Success(t *testing.T) {
	resp := model.GempaListResponse{}
	resp.Infogempa.Gempa = []model.Earthquake{
		{Magnitude: "5.1", Depth: "10 km", Region: "Test 1"},
		{Magnitude: "5.2", Depth: "10 km", Region: "Test 2"},
	}

	srv := mockServer(t, http.StatusOK, resp)
	defer srv.Close()

	cfg := config.BMKGConfig{
		BaseURL:     srv.URL,
		HTTPTimeout: 5 * time.Second,
		MaxRetries:  0,
	}
	client := New(cfg)

	list, err := client.FetchM5Plus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}
}

func TestFetchFelt_Success(t *testing.T) {
	resp := model.GempaListResponse{}
	resp.Infogempa.Gempa = []model.Earthquake{
		{Magnitude: "3.2", Depth: "5 km", Region: "Felt test", Felt: "II Sigi"},
	}

	srv := mockServer(t, http.StatusOK, resp)
	defer srv.Close()

	cfg := config.BMKGConfig{
		BaseURL:     srv.URL,
		HTTPTimeout: 5 * time.Second,
		MaxRetries:  0,
	}
	client := New(cfg)

	list, err := client.FetchFelt()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 item, got %d", len(list))
	}
	if list[0].Felt != "II Sigi" {
		t.Errorf("expected felt 'II Sigi', got %s", list[0].Felt)
	}
}

func TestShakemapURL(t *testing.T) {
	cfg := config.BMKGConfig{
		ShakemapURL: "https://static.bmkg.go.id",
	}
	client := New(cfg)

	url := client.ShakemapURL("20260619085320.mmi.jpg")
	expected := "https://static.bmkg.go.id/20260619085320.mmi.jpg"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestBuildURL(t *testing.T) {
	c := &client{
		baseURL:  "https://data.bmkg.go.id",
		jsonPath: "/DataMKG/TEWS",
	}
	url := c.buildURL("/autogempa.json")
	expected := "https://data.bmkg.go.id/DataMKG/TEWS/autogempa.json"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestFetchLatest_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.BMKGConfig{
		BaseURL:     srv.URL,
		HTTPTimeout: 5 * time.Second,
		MaxRetries:  0,
	}
	client := New(cfg)

	_, err := client.FetchLatest()
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
}

func TestFetchLatest_NotFound(t *testing.T) {
	srv := mockServer(t, http.StatusNotFound, nil)
	defer srv.Close()

	cfg := config.BMKGConfig{
		BaseURL:     srv.URL,
		HTTPTimeout: 5 * time.Second,
		MaxRetries:  0,
	}
	client := New(cfg)

	_, err := client.FetchLatest()
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("expected 'short', got %s", got)
	}
	if got := truncate("this is a long string", 10); got != "this is a ..." {
		t.Errorf("expected 'this is a ...', got %s", got)
	}
	if got := truncate("", 5); got != "" {
		t.Errorf("expected '', got %s", got)
	}
}

// isErrRateLimited checks if the error chain contains a rate limit error.
func isErrRateLimited(err error) bool {
	return err != nil && strings.Contains(err.Error(), "rate limit exceeded")
}
