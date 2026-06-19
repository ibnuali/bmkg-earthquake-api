package service

import (
	"errors"
	"testing"
	"time"

	"earthquake-api/internal/config"
	"earthquake-api/internal/model"
)

// mockBMKG implements bmkg.Client for testing.
type mockBMKG struct {
	latestFunc func() (*model.Earthquake, error)
	m5Func     func() ([]model.Earthquake, error)
	feltFunc   func() ([]model.Earthquake, error)
	shakemap   string
}

func (m *mockBMKG) FetchLatest() (*model.Earthquake, error) {
	if m.latestFunc != nil {
		return m.latestFunc()
	}
	return nil, errors.New("not mocked")
}

func (m *mockBMKG) FetchM5Plus() ([]model.Earthquake, error) {
	if m.m5Func != nil {
		return m.m5Func()
	}
	return nil, errors.New("not mocked")
}

func (m *mockBMKG) FetchFelt() ([]model.Earthquake, error) {
	if m.feltFunc != nil {
		return m.feltFunc()
	}
	return nil, errors.New("not mocked")
}

func (m *mockBMKG) ShakemapURL(code string) string {
	return "https://static.bmkg.go.id/" + code
}

// ---- ParseEarthquake Tests (existing + expanded) ----

func TestParseEarthquake(t *testing.T) {
	raw := &model.Earthquake{
		Date:        "19 Jun 2026",
		Time:        "08:53:20 WIB",
		DateTime:    "2026-06-19T01:53:20+00:00",
		Coordinates: "-1.17,120.01",
		Magnitude:   "3.3",
		Depth:       "4 km",
		Region:      "Pusat gempa berada di darat 28 km timur laut Sigi",
		Potency:     "Tidak berpotensi tsunami",
		Shakemap:    "20260619085320.mmi.jpg",
	}

	parsed, err := parseEarthquake(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Date != "19 Jun 2026" {
		t.Errorf("expected date '19 Jun 2026', got %s", parsed.Date)
	}
	if parsed.Magnitude != 3.3 {
		t.Errorf("expected magnitude 3.3, got %f", parsed.Magnitude)
	}
	if parsed.DepthKM != 4 {
		t.Errorf("expected depth 4, got %f", parsed.DepthKM)
	}
	if parsed.Latitude != -1.17 {
		t.Errorf("expected latitude -1.17, got %f", parsed.Latitude)
	}
	if parsed.Longitude != 120.01 {
		t.Errorf("expected longitude 120.01, got %f", parsed.Longitude)
	}
	if parsed.ShakemapURL != "https://static.bmkg.go.id/20260619085320.mmi.jpg" {
		t.Errorf("unexpected shakemap URL: %s", parsed.ShakemapURL)
	}
	if len(parsed.Coordinates) != 2 {
		t.Errorf("expected 2 coordinates, got %d", len(parsed.Coordinates))
	}
}

func TestParseEarthquakeWithFelt(t *testing.T) {
	raw := &model.Earthquake{
		Date:     "19 Jun 2026",
		Time:     "03:29:27 WIB",
		DateTime: "2026-06-18T20:29:27+00:00",
		Magnitude: "4.0",
		Depth:    "10 km",
		Region:   "Pusat gempa berada di darat 43 km timur laut Sigi",
		Felt:     "III Palu, III Sigi",
	}
	parsed, err := parseEarthquake(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Felt != "III Palu, III Sigi" {
		t.Errorf("expected felt 'III Palu, III Sigi', got %s", parsed.Felt)
	}
}

func TestParseEarthquakeEmptyShakemap(t *testing.T) {
	raw := &model.Earthquake{
		Date:      "19 Jun 2026",
		Time:      "08:53:20 WIB",
		DateTime:  "2026-06-19T01:53:20+00:00",
		Magnitude: "2.5",
		Depth:     "10 km",
		Region:    "Test region",
	}
	parsed, err := parseEarthquake(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.ShakemapURL != "" {
		t.Errorf("expected empty shakemap URL, got %s", parsed.ShakemapURL)
	}
}

func TestParseEarthquakeInvalidCoordinates(t *testing.T) {
	raw := &model.Earthquake{
		Date:        "19 Jun 2026",
		Time:        "08:53:20 WIB",
		DateTime:    "2026-06-19T01:53:20+00:00",
		Coordinates: "invalid",
		Magnitude:   "3.3",
		Depth:       "4 km",
		Region:      "Test region",
	}
	parsed, err := parseEarthquake(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Latitude != 0 || parsed.Longitude != 0 {
		t.Errorf("expected 0 coordinates on parse failure, got %f, %f", parsed.Latitude, parsed.Longitude)
	}
}

// ---- Additional parseEarthquake edge cases ----

func TestParseEarthquake_InvalidMagnitude(t *testing.T) {
	raw := &model.Earthquake{
		DateTime:  "2026-06-19T01:53:20+00:00",
		Magnitude: "not-a-number",
		Depth:     "10 km",
		Region:    "Test",
	}
	parsed, err := parseEarthquake(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Magnitude != 0 {
		t.Errorf("expected 0 for invalid magnitude, got %f", parsed.Magnitude)
	}
}

func TestParseEarthquake_InvalidDepth(t *testing.T) {
	raw := &model.Earthquake{
		DateTime:  "2026-06-19T01:53:20+00:00",
		Magnitude: "3.0",
		Depth:     "not-a-number KM",
		Region:    "Test",
	}
	parsed, err := parseEarthquake(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.DepthKM != 0 {
		t.Errorf("expected 0 for invalid depth, got %f", parsed.DepthKM)
	}
}

func TestParseEarthquake_DifferentDepthFormats(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"10 km", 10},
		{"5 Km", 5},
		{"112 km", 112},
		{"0 km", 0},
		{"", 0},
	}
	for _, tt := range tests {
		raw := &model.Earthquake{
			DateTime:  "2026-06-19T01:53:20+00:00",
			Magnitude: "3.0",
			Depth:     tt.input,
			Region:    "Test",
		}
		parsed, _ := parseEarthquake(raw)
		if parsed.DepthKM != tt.want {
			t.Errorf("parseDepth(%q) = %f, want %f", tt.input, parsed.DepthKM, tt.want)
		}
	}
}

func TestParseEarthquake_DateTimeFallback(t *testing.T) {
	// Invalid DateTime should fall back to current time
	raw := &model.Earthquake{
		DateTime:  "invalid-date",
		Magnitude: "3.0",
		Depth:     "10 km",
		Region:    "Test",
	}
	parsed, err := parseEarthquake(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be close to now (within 5 seconds)
	if time.Since(parsed.DateTime) > 5*time.Second {
		t.Errorf("expected datetime near now, got %v (diff: %v)", parsed.DateTime, time.Since(parsed.DateTime))
	}
}

func TestParseEarthquake_ShakemapFullURL(t *testing.T) {
	raw := &model.Earthquake{
		DateTime:  "2026-06-19T01:53:20+00:00",
		Magnitude: "3.0",
		Depth:     "10 km",
		Region:    "Test",
		Shakemap:  "https://cdn.example.com/shakemap.jpg",
	}
	parsed, err := parseEarthquake(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.ShakemapURL != "https://cdn.example.com/shakemap.jpg" {
		t.Errorf("expected full URL preserved, got %s", parsed.ShakemapURL)
	}
}

func TestParseEarthquakeList(t *testing.T) {
	list := []model.Earthquake{
		{
			Date:      "19 Jun 2026",
			Time:      "08:53:20 WIB",
			DateTime:  "2026-06-19T01:53:20+00:00",
			Magnitude: "3.3",
			Depth:     "4 km",
			Region:    "Test region 1",
		},
		{
			Date:      "18 Jun 2026",
			Time:      "13:57:32 WIB",
			DateTime:  "2026-06-18T06:57:32+00:00",
			Magnitude: "5.5",
			Depth:     "10 km",
			Region:    "Test region 2",
		},
	}
	parsed, err := parseEarthquakeList(list)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 items, got %d", len(parsed))
	}
}

func TestParseEarthquakeList_Empty(t *testing.T) {
	parsed, err := parseEarthquakeList([]model.Earthquake{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed) != 0 {
		t.Errorf("expected 0 items, got %d", len(parsed))
	}
}

func TestParseEarthquakeList_SkipMalformed(t *testing.T) {
	list := []model.Earthquake{
		{
			Date:      "19 Jun 2026",
			Time:      "08:53:20 WIB",
			DateTime:  "2026-06-19T01:53:20+00:00",
			Magnitude: "3.3",
			Depth:     "4 km",
			Region:    "Test region 1",
		},
		// This entry will still parse (fields other than datetime don't cause errors)
		{
			DateTime:  "2026-06-19T01:53:20+00:00",
			Region:    "Test region 2",
		},
	}
	parsed, err := parseEarthquakeList(list)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 items (both parse safely), got %d", len(parsed))
	}
}

// ---- Service with caching tests ----

func TestNewEarthquakeService_WithCache(t *testing.T) {
	svc := NewEarthquakeService(&mockBMKG{}, config.CacheConfig{
		Enabled:     true,
		TTL:         time.Minute,
		CleanupIntv: time.Minute,
	})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if !svc.cacheEnabled {
		t.Error("expected cache enabled")
	}
	if svc.cache == nil {
		t.Error("expected non-nil cache")
	}
}

func TestNewEarthquakeService_WithoutCache(t *testing.T) {
	svc := NewEarthquakeService(&mockBMKG{}, config.CacheConfig{
		Enabled: false,
	})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.cacheEnabled {
		t.Error("expected cache disabled")
	}
	if svc.cache != nil {
		t.Error("expected nil cache when disabled")
	}
}

func TestGetLatest_WithCache(t *testing.T) {
	callCount := 0
	mock := &mockBMKG{
		latestFunc: func() (*model.Earthquake, error) {
			callCount++
			return &model.Earthquake{
				DateTime:  "2026-06-19T01:53:20+00:00",
				Magnitude: "3.3",
				Depth:     "4 km",
				Region:    "Test",
			}, nil
		},
	}
	svc := NewEarthquakeService(mock, config.CacheConfig{
		Enabled: true,
		TTL:     time.Minute,
	})

	// First call — should hit client
	result1, err := svc.GetLatest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result1 == nil {
		t.Fatal("expected non-nil result")
	}
	if result1.Magnitude != 3.3 {
		t.Errorf("expected magnitude 3.3, got %f", result1.Magnitude)
	}

	// Second call — should hit cache
	result2, err := svc.GetLatest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2 == nil {
		t.Fatal("expected non-nil result")
	}

	if callCount != 1 {
		t.Errorf("expected 1 client call due to caching, got %d", callCount)
	}
}

func TestGetLatest_WithoutCache(t *testing.T) {
	callCount := 0
	mock := &mockBMKG{
		latestFunc: func() (*model.Earthquake, error) {
			callCount++
			return &model.Earthquake{
				DateTime:  "2026-06-19T01:53:20+00:00",
				Magnitude: "5.0",
				Depth:     "10 km",
				Region:    "Test",
			}, nil
		},
	}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: false})

	// Both calls should hit the client
	svc.GetLatest()
	svc.GetLatest()

	if callCount != 2 {
		t.Errorf("expected 2 client calls (no cache), got %d", callCount)
	}
}

func TestGetLatest_ClientError(t *testing.T) {
	mock := &mockBMKG{
		latestFunc: func() (*model.Earthquake, error) {
			return nil, model.ErrUpstream
		},
	}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: false})

	_, err := svc.GetLatest()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetM5Plus_Success(t *testing.T) {
	mock := &mockBMKG{
		m5Func: func() ([]model.Earthquake, error) {
			return []model.Earthquake{
				{DateTime: "2026-06-19T01:53:20+00:00", Magnitude: "5.5", Depth: "10 km", Region: "A"},
				{DateTime: "2026-06-18T06:57:32+00:00", Magnitude: "5.0", Depth: "10 km", Region: "B"},
			}, nil
		},
	}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: false})

	list, err := svc.GetM5Plus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}
}

func TestGetM5Plus_WithCache(t *testing.T) {
	callCount := 0
	mock := &mockBMKG{
		m5Func: func() ([]model.Earthquake, error) {
			callCount++
			return []model.Earthquake{
				{DateTime: "2026-06-19T01:53:20+00:00", Magnitude: "5.0", Depth: "10 km", Region: "A"},
			}, nil
		},
	}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: true, TTL: time.Minute})

	svc.GetM5Plus()
	svc.GetM5Plus()

	if callCount != 1 {
		t.Errorf("expected 1 client call due to caching, got %d", callCount)
	}
}

func TestGetM5Plus_Error(t *testing.T) {
	mock := &mockBMKG{
		m5Func: func() ([]model.Earthquake, error) {
			return nil, model.ErrRateLimited
		},
	}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: false})

	_, err := svc.GetM5Plus()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetFelt_Success(t *testing.T) {
	mock := &mockBMKG{
		feltFunc: func() ([]model.Earthquake, error) {
			return []model.Earthquake{
				{DateTime: "2026-06-19T01:53:20+00:00", Magnitude: "3.2", Depth: "5 km", Region: "A", Felt: "II Sigi"},
			}, nil
		},
	}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: false})

	list, err := svc.GetFelt()
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

func TestGetFelt_WithCache(t *testing.T) {
	callCount := 0
	mock := &mockBMKG{
		feltFunc: func() ([]model.Earthquake, error) {
			callCount++
			return []model.Earthquake{
				{DateTime: "2026-06-19T01:53:20+00:00", Magnitude: "3.0", Depth: "5 km", Region: "A", Felt: "II"},
			}, nil
		},
	}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: true, TTL: time.Minute})

	svc.GetFelt()
	svc.GetFelt()

	if callCount != 1 {
		t.Errorf("expected 1 client call due to caching, got %d", callCount)
	}
}

func TestGetFelt_Error(t *testing.T) {
	mock := &mockBMKG{
		feltFunc: func() ([]model.Earthquake, error) {
			return nil, model.ErrNotFound
		},
	}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: false})

	_, err := svc.GetFelt()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetShakemapURL_Success(t *testing.T) {
	mock := &mockBMKG{}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: false})

	url, err := svc.GetShakemapURL("20260619085320.mmi.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://static.bmkg.go.id/20260619085320.mmi.jpg"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestGetShakemapURL_InvalidCode(t *testing.T) {
	mock := &mockBMKG{}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: false})

	_, err := svc.GetShakemapURL("")
	if err == nil {
		t.Fatal("expected error for empty code, got nil")
	}
}

func TestGetShakemapURL_MaliciousCode(t *testing.T) {
	mock := &mockBMKG{}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: false})

	_, err := svc.GetShakemapURL("<script>alert(1)</script>")
	if err == nil {
		t.Fatal("expected error for malicious code, got nil")
	}
}

func TestGetShakemapURL_URLEndodedCode(t *testing.T) {
	mock := &mockBMKG{}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: false})

	_, err := svc.GetShakemapURL("%2E%2E%2F")
	if err == nil {
		t.Fatal("expected error for URL-encoded code, got nil")
	}
}

// ---- Cross-type Cache Isolation ----

func TestCacheKeysAreIsolated(t *testing.T) {
	latestCallCount := 0
	m5CallCount := 0
	mock := &mockBMKG{
		latestFunc: func() (*model.Earthquake, error) {
			latestCallCount++
			return &model.Earthquake{DateTime: "2026-06-19T01:53:20+00:00", Magnitude: "3.3", Depth: "4 km", Region: "Latest"}, nil
		},
		m5Func: func() ([]model.Earthquake, error) {
			m5CallCount++
			return []model.Earthquake{{DateTime: "2026-06-18T06:57:32+00:00", Magnitude: "5.0", Depth: "10 km", Region: "M5"}}, nil
		},
	}
	svc := NewEarthquakeService(mock, config.CacheConfig{Enabled: true, TTL: time.Minute})

	// Fetch latest (cached)
	svc.GetLatest()
	svc.GetLatest()
	// Fetch M5+ (cached separately)
	svc.GetM5Plus()
	svc.GetM5Plus()

	if latestCallCount != 1 {
		t.Errorf("expected 1 latest call, got %d", latestCallCount)
	}
	if m5CallCount != 1 {
		t.Errorf("expected 1 m5+ call, got %d", m5CallCount)
	}
}
