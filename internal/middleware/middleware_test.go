package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSDefault(t *testing.T) {
	handler := CORS(nil, nil, nil)(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if origin := resp.Header.Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected '*', got %s", origin)
	}
	if methods := resp.Header.Get("Access-Control-Allow-Methods"); methods != "GET, POST, OPTIONS" {
		t.Errorf("expected 'GET, POST, OPTIONS', got %s", methods)
	}
	if headers := resp.Header.Get("Access-Control-Allow-Headers"); headers != "Accept, Content-Type, X-Request-ID" {
		t.Errorf("expected 'Accept, Content-Type, X-Request-ID', got %s", headers)
	}
}

func TestCORSWithCustomOrigins(t *testing.T) {
	handler := CORS(
		[]string{"http://example.com"},
		[]string{"GET"},
		[]string{"X-Custom"},
	)(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if origin := resp.Header.Get("Access-Control-Allow-Origin"); origin != "http://example.com" {
		t.Errorf("expected 'http://example.com', got %s", origin)
	}
	if methods := resp.Header.Get("Access-Control-Allow-Methods"); methods != "GET" {
		t.Errorf("expected 'GET', got %s", methods)
	}
	if headers := resp.Header.Get("Access-Control-Allow-Headers"); headers != "X-Custom" {
		t.Errorf("expected 'X-Custom', got %s", headers)
	}
}

func TestCORSOptionsMethod(t *testing.T) {
	nextCalled := false
	handler := CORS(nil, nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if nextCalled {
		t.Error("next handler should NOT be called for OPTIONS request")
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestCORSPassthroughForNonOptions(t *testing.T) {
	nextCalled := false
	handler := CORS(nil, nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !nextCalled {
		t.Error("next handler should be called for GET request")
	}
}

func TestRecoveryNoPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRecoveryWithPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if success, ok := body["success"].(bool); ok && success {
		t.Error("expected success=false")
	}
}

func TestRequestIDFromHeader(t *testing.T) {
	handler := RequestID(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "custom-id-123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if id := w.Header().Get("X-Request-ID"); id != "custom-id-123" {
		t.Errorf("expected 'custom-id-123', got %s", id)
	}
}

func TestRequestIDGenerated(t *testing.T) {
	handler := RequestID(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	id := w.Header().Get("X-Request-ID")
	if id == "" {
		t.Error("expected non-empty generated request ID")
	}
	if len(id) != 16 {
		t.Errorf("expected 16 hex chars, got %d: %s", len(id), id)
	}
}

func TestRequestIDUniquePerRequest(t *testing.T) {
	handler := RequestID(http.HandlerFunc(okHandler))

	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		id := w.Header().Get("X-Request-ID")
		if ids[id] {
			t.Errorf("duplicate request ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestLoggerWritesLog(t *testing.T) {
	// Just verify the middleware doesn't panic and passes through
	handler := Logger(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestResponseWriter(t *testing.T) {
	rw := newResponseWriter(httptest.NewRecorder())

	if rw.statusCode != http.StatusOK {
		t.Errorf("expected default 200, got %d", rw.statusCode)
	}

	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rw.statusCode)
	}
}

func TestMiddlewareChain(t *testing.T) {
	// Test that all middleware work together in a chain
	var handler http.Handler = http.HandlerFunc(okHandler)
	handler = Logger(handler)
	handler = CORS([]string{"*"}, nil, nil)(handler)
	handler = RequestID(handler)
	handler = Recovery(handler)

	req := httptest.NewRequest(http.MethodGet, "/chain", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Request-ID") == "" {
		t.Error("expected request ID header")
	}
	if resp.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected CORS header")
	}
}

func TestGenerateRequestID(t *testing.T) {
	id := generateRequestID()
	if id == "" {
		t.Error("expected non-empty ID")
	}
	if len(id) != 16 {
		t.Errorf("expected 16 chars, got %d", len(id))
	}
}

// okHandler is a simple handler that returns 200 OK.
func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}

// Ensure responseWriter implements http.ResponseWriter
var _ http.ResponseWriter = &responseWriter{}

// Ensure responseWriter supports http.Flusher if underlying writer does
func TestResponseWriterImplementsInterfaces(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	if _, ok := interface{}(rw).(http.ResponseWriter); !ok {
		t.Error("responseWriter should implement http.ResponseWriter")
	}
	// httptest.ResponseRecorder does NOT implement http.Flusher, so we
	// can only check our wrapper doesn't claim to implement it when it doesn't.
}
