package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewSuccess(t *testing.T) {
	resp := NewSuccess(map[string]string{"key": "value"})

	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Error != nil {
		t.Errorf("expected nil error, got %v", resp.Error)
	}
	if resp.Metadata.Source != "BMKG" {
		t.Errorf("expected source BMKG, got %s", resp.Metadata.Source)
	}
	if resp.Metadata.APIVersion != "v1" {
		t.Errorf("expected api_version v1, got %s", resp.Metadata.APIVersion)
	}

	data, ok := resp.Data.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T", resp.Data)
	}
	if data["key"] != "value" {
		t.Errorf("expected 'value', got %s", data["key"])
	}
}

func TestNewSuccessNilData(t *testing.T) {
	resp := NewSuccess(nil)
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Data != nil {
		t.Errorf("expected nil data, got %v", resp.Data)
	}
}

func TestNewError(t *testing.T) {
	resp := NewError("NOT_FOUND", "data not found")

	if resp.Success {
		t.Error("expected success=false")
	}
	if resp.Data != nil {
		t.Errorf("expected nil data, got %v", resp.Data)
	}
	if resp.Error == nil {
		t.Fatal("expected non-nil error")
	}
	if resp.Error.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %s", resp.Error.Code)
	}
	if resp.Error.Message != "data not found" {
		t.Errorf("expected message 'data not found', got %s", resp.Error.Message)
	}
}

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, NewSuccess("ok"))

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %s", ct)
	}

	var body APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if !body.Success {
		t.Error("expected success=true")
	}
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	Success(w, map[string]int{"count": 42})

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if !body.Success {
		t.Error("expected success=true")
	}

	dataMap, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", body.Data)
	}
	if count, ok := dataMap["count"].(float64); !ok || count != 42 {
		t.Errorf("expected count=42, got %v", dataMap["count"])
	}
}

func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	Created(w, "created")

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid parameter")

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	var body APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body.Success {
		t.Error("expected success=false")
	}
	if body.Error.Code != "BAD_REQUEST" {
		t.Errorf("expected BAD_REQUEST, got %s", body.Error.Code)
	}
	if body.Error.Message != "invalid parameter" {
		t.Errorf("expected 'invalid parameter', got %s", body.Error.Message)
	}
}

func TestErrorFrom(t *testing.T) {
	w := httptest.NewRecorder()
	ErrorFrom(w, http.StatusServiceUnavailable, http.ErrAbortHandler)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}

	var body APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body.Error.Code != "UPSTREAM_ERROR" {
		t.Errorf("expected UPSTREAM_ERROR, got %s", body.Error.Code)
	}
}

func TestMetadataTimestamp(t *testing.T) {
	before := time.Now().UTC()
	resp := NewSuccess("data")
	after := time.Now().UTC()

	ts := resp.Metadata.Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v should be between %v and %v", ts, before, after)
	}
}
