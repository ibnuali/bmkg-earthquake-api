package response

import (
	"encoding/json"
	"net/http"
	"time"
)

// APIError represents an error response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Metadata contains metadata about the response.
type Metadata struct {
	Source     string    `json:"source"`
	APIVersion string    `json:"api_version"`
	Timestamp  time.Time `json:"timestamp"`
}

// APIResponse is the standard JSON envelope for all API responses.
type APIResponse struct {
	Success  bool        `json:"success"`
	Data     interface{} `json:"data"`
	Error    *APIError   `json:"error"`
	Metadata Metadata    `json:"metadata"`
}

// NewSuccess creates a success API response.
func NewSuccess(data interface{}) APIResponse {
	return APIResponse{
		Success: true,
		Data:    data,
		Error:   nil,
		Metadata: Metadata{
			Source:     "BMKG",
			APIVersion: "v1",
			Timestamp:  time.Now().UTC(),
		},
	}
}

// NewError creates an error API response.
func NewError(code, message string) APIResponse {
	return APIResponse{
		Success: false,
		Data:    nil,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
		Metadata: Metadata{
			Source:     "BMKG",
			APIVersion: "v1",
			Timestamp:  time.Now().UTC(),
		},
	}
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Best-effort: if we can't encode the response, it's a 500
		http.Error(w, `{"success":false,"error":{"code":"INTERNAL_ERROR","message":"failed to encode response"}}`, http.StatusInternalServerError)
	}
}

// Success writes a 200 success JSON response.
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, NewSuccess(data))
}

// Created writes a 201 success JSON response.
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, NewSuccess(data))
}

// Error writes an error JSON response with the given status code.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, NewError(code, message))
}

// ErrorFrom writes an error JSON response from an existing error.
func ErrorFrom(w http.ResponseWriter, status int, err error) {
	Error(w, status, "UPSTREAM_ERROR", err.Error())
}
