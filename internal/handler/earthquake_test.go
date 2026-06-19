package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"earthquake-api/internal/config"
	"earthquake-api/internal/model"
	"earthquake-api/internal/service"
)

// mockBMKGClient implements bmkg.Client for testing.
type mockBMKGClient struct {
	latestFunc  func() (*model.Earthquake, error)
	m5PlusFunc  func() ([]model.Earthquake, error)
	feltFunc    func() ([]model.Earthquake, error)
	shakemapURL string
}

func (m *mockBMKGClient) FetchLatest() (*model.Earthquake, error) {
	if m.latestFunc != nil {
		return m.latestFunc()
	}
	return defaultLatest(), nil
}

func (m *mockBMKGClient) FetchM5Plus() ([]model.Earthquake, error) {
	if m.m5PlusFunc != nil {
		return m.m5PlusFunc()
	}
	return defaultList(), nil
}

func (m *mockBMKGClient) FetchFelt() ([]model.Earthquake, error) {
	if m.feltFunc != nil {
		return m.feltFunc()
	}
	return defaultList(), nil
}

func (m *mockBMKGClient) ShakemapURL(code string) string {
	if m.shakemapURL != "" {
		return m.shakemapURL + "/" + code
	}
	return "https://static.bmkg.go.id/" + code
}

func defaultLatest() *model.Earthquake {
	return &model.Earthquake{
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
}

func defaultList() []model.Earthquake {
	return []model.Earthquake{
		{
			Date:      "19 Jun 2026",
			Time:      "08:53:20 WIB",
			DateTime:  "2026-06-19T01:53:20+00:00",
			Magnitude: "3.3",
			Depth:     "4 km",
			Region:    "Gempa test 1",
		},
		{
			Date:      "18 Jun 2026",
			Time:      "13:57:32 WIB",
			DateTime:  "2026-06-18T06:57:32+00:00",
			Magnitude: "5.5",
			Depth:     "10 km",
			Region:    "Gempa test 2",
		},
	}
}

func newTestHandler(mock *mockBMKGClient) *EarthquakeHandler {
	svc := service.NewEarthquakeService(mock, config.CacheConfig{Enabled: false})
	return NewEarthquakeHandler(svc)
}

func newTestHandlerWithCache(mock *mockBMKGClient) *EarthquakeHandler {
	svc := service.NewEarthquakeService(mock, config.CacheConfig{
		Enabled:     true,
		TTL:         time.Minute,
		CleanupIntv: time.Minute,
	})
	return NewEarthquakeHandler(svc)
}

// decodeResponse decodes the standard API response from a recorder.
func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return body
}

// --- Health ---

func TestHealth(t *testing.T) {
	h := newTestHandler(&mockBMKGClient{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	h.Health(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := decodeResponse(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	if data["status"] != "healthy" {
		t.Errorf("expected status=healthy, got %v", data["status"])
	}
}

// --- Home ---

func TestHome(t *testing.T) {
	h := newTestHandler(&mockBMKGClient{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h.Home(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := decodeResponse(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	if data["name"] != "BMKG Earthquake API" {
		t.Errorf("unexpected name: %v", data["name"])
	}
	endpoints, ok := data["endpoints"].(map[string]interface{})
	if !ok {
		t.Fatal("expected endpoints map")
	}
	if _, ok := endpoints["GET /health"]; !ok {
		t.Error("expected /health in endpoints")
	}
}

// --- GetLatest ---

func TestGetLatest_Success(t *testing.T) {
	h := newTestHandler(&mockBMKGClient{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/latest", nil)
	h.GetLatest(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := decodeResponse(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}

	data := body["data"].(map[string]interface{})
	if data["magnitude"] != 3.3 {
		t.Errorf("expected magnitude 3.3, got %v", data["magnitude"])
	}
	if data["region"] != "Pusat gempa berada di darat 28 km timur laut Sigi" {
		t.Errorf("unexpected region: %v", data["region"])
	}
}

func TestGetLatest_UpstreamError(t *testing.T) {
	mock := &mockBMKGClient{
		latestFunc: func() (*model.Earthquake, error) {
			return nil, errors.New("connection refused")
		},
	}
	h := newTestHandler(mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/latest", nil)
	h.GetLatest(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}

	body := decodeResponse(t, w)
	if body["success"] != false {
		t.Error("expected success=false")
	}
	err, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object")
	}
	if err["code"] != "UPSTREAM_ERROR" {
		t.Errorf("expected UPSTREAM_ERROR, got %s", err["code"])
	}
}

func TestGetLatest_RateLimited(t *testing.T) {
	mock := &mockBMKGClient{
		latestFunc: func() (*model.Earthquake, error) {
			return nil, model.ErrRateLimited
		},
	}
	h := newTestHandler(mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/latest", nil)
	h.GetLatest(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}

	body := decodeResponse(t, w)
	err := body["error"].(map[string]interface{})
	if err["code"] != "UPSTREAM_RATE_LIMITED" {
		t.Errorf("expected UPSTREAM_RATE_LIMITED, got %s", err["code"])
	}
}

// --- GetM5Plus ---

func TestGetM5Plus_Success(t *testing.T) {
	h := newTestHandler(&mockBMKGClient{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/list/magnitude5", nil)
	h.GetM5Plus(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := decodeResponse(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("expected 2 items, got %d", len(data))
	}
}

func TestGetM5Plus_Empty(t *testing.T) {
	mock := &mockBMKGClient{
		m5PlusFunc: func() ([]model.Earthquake, error) {
			return []model.Earthquake{}, nil
		},
	}
	h := newTestHandler(mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/list/magnitude5", nil)
	h.GetM5Plus(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := decodeResponse(t, w)
	data := body["data"].([]interface{})
	if len(data) != 0 {
		t.Errorf("expected 0 items, got %d", len(data))
	}
}

func TestGetM5Plus_UpstreamError(t *testing.T) {
	mock := &mockBMKGClient{
		m5PlusFunc: func() ([]model.Earthquake, error) {
			return nil, errors.New("upstream unavailable")
		},
	}
	h := newTestHandler(mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/list/magnitude5", nil)
	h.GetM5Plus(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// --- GetFelt ---

func TestGetFelt_Success(t *testing.T) {
	h := newTestHandler(&mockBMKGClient{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/list/felt", nil)
	h.GetFelt(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := decodeResponse(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

func TestGetFelt_WithFeltField(t *testing.T) {
	mock := &mockBMKGClient{
		feltFunc: func() ([]model.Earthquake, error) {
			return []model.Earthquake{
				{
					Date:     "19 Jun 2026",
					Time:     "03:29:27 WIB",
					DateTime: "2026-06-18T20:29:27+00:00",
					Magnitude: "4.0",
					Depth:    "10 km",
					Region:   "Pusat gempa di darat",
					Felt:     "III Palu, III Sigi",
				},
			}, nil
		},
	}
	h := newTestHandler(mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/list/felt", nil)
	h.GetFelt(w, r)

	body := decodeResponse(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].([]interface{})
	item := data[0].(map[string]interface{})
	if item["felt"] != "III Palu, III Sigi" {
		t.Errorf("expected felt, got %v", item["felt"])
	}
}

func TestGetFelt_NotFound(t *testing.T) {
	mock := &mockBMKGClient{
		feltFunc: func() ([]model.Earthquake, error) {
			return nil, model.ErrNotFound
		},
	}
	h := newTestHandler(mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/list/felt", nil)
	h.GetFelt(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- GetShakemap ---

func TestGetShakemap_Success(t *testing.T) {
	h := newTestHandler(&mockBMKGClient{shakemapURL: "https://static.bmkg.go.id"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/shakemap?code=20260619085320.mmi.jpg", nil)
	h.GetShakemap(w, r)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	expected := "https://static.bmkg.go.id/20260619085320.mmi.jpg"
	if location != expected {
		t.Errorf("expected location %s, got %s", expected, location)
	}
}

func TestGetShakemap_MissingCode(t *testing.T) {
	h := newTestHandler(&mockBMKGClient{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/shakemap", nil)
	h.GetShakemap(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	body := decodeResponse(t, w)
	err := body["error"].(map[string]interface{})
	if err["code"] != "INVALID_PARAMETER" {
		t.Errorf("expected INVALID_PARAMETER, got %s", err["code"])
	}
}

func TestGetShakemap_InvalidCode(t *testing.T) {
	h := newTestHandler(&mockBMKGClient{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/shakemap?code=<script>", nil)
	h.GetShakemap(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- APIResponse envelope structure tests ---

func TestAPISuccessResponseStructure(t *testing.T) {
	h := newTestHandler(&mockBMKGClient{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/latest", nil)
	h.GetLatest(w, r)

	body := decodeResponse(t, w)

	// Verify standard envelope fields exist
	checkFields := []string{"success", "data", "error", "metadata"}
	for _, field := range checkFields {
		if _, exists := body[field]; !exists {
			t.Errorf("missing field: %s", field)
		}
	}

	metadata := body["metadata"].(map[string]interface{})
	metaFields := []string{"source", "api_version", "timestamp"}
	for _, field := range metaFields {
		if _, exists := metadata[field]; !exists {
			t.Errorf("missing metadata field: %s", field)
		}
	}

	if metadata["source"] != "BMKG" {
		t.Errorf("expected source BMKG, got %s", metadata["source"])
	}
	if metadata["api_version"] != "v1" {
		t.Errorf("expected api_version v1, got %s", metadata["api_version"])
	}
}

func TestAPIErrorResponseStructure(t *testing.T) {
	mock := &mockBMKGClient{
		latestFunc: func() (*model.Earthquake, error) {
			return nil, errors.New("upstream error")
		},
	}
	h := newTestHandler(mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/latest", nil)
	h.GetLatest(w, r)

	body := decodeResponse(t, w)

	checkFields := []string{"success", "data", "error", "metadata"}
	for _, field := range checkFields {
		if _, exists := body[field]; !exists {
			t.Errorf("missing field: %s", field)
		}
	}

	if body["data"] != nil {
		t.Error("expected nil data on error")
	}

	err := body["error"].(map[string]interface{})
	if _, exists := err["code"]; !exists {
		t.Error("missing error.code")
	}
	if _, exists := err["message"]; !exists {
		t.Error("missing error.message")
	}
}

// --- writeError edge cases ---

func TestWriteError_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, model.ErrNotFound)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestWriteError_RateLimited(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, model.ErrRateLimited)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}

func TestWriteError_InvalidRequest(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, model.ErrInvalidRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWriteError_Default(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, errors.New("some random error"))

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// --- Caching integration ---

func TestLatestCaching(t *testing.T) {
	callCount := 0
	mock := &mockBMKGClient{
		latestFunc: func() (*model.Earthquake, error) {
			callCount++
			return defaultLatest(), nil
		},
	}
	h := newTestHandlerWithCache(mock)

	// First call — should hit the real client
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/latest", nil)
	h.GetLatest(w1, r1)
	if w1.Code != http.StatusOK {
		t.Errorf("first call: expected 200, got %d", w1.Code)
	}

	// Second call — should hit cache
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/latest", nil)
	h.GetLatest(w2, r2)
	if w2.Code != http.StatusOK {
		t.Errorf("second call: expected 200, got %d", w2.Code)
	}

	if callCount != 1 {
		t.Errorf("expected 1 client call due to caching, got %d", callCount)
	}
}

// --- JSON Content-Type verification ---

func TestContentTypeIsJSON(t *testing.T) {
	h := newTestHandler(&mockBMKGClient{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/earthquake/latest", nil)
	h.GetLatest(w, r)

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected application/json, got %s", ct)
	}
}
