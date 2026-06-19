package model

import "testing"

func TestErrorsAreDefined(t *testing.T) {
	tests := []struct {
		err   error
		msg   string
	}{
		{ErrNotFound, "data not found"},
		{ErrUpstream, "upstream service error"},
		{ErrRateLimited, "rate limit exceeded"},
		{ErrInvalidRequest, "invalid request"},
	}

	for _, tt := range tests {
		if tt.err == nil {
			t.Error("error should not be nil")
		}
		if tt.err.Error() != tt.msg {
			t.Errorf("expected %q, got %q", tt.msg, tt.err.Error())
		}
	}
}

func TestEarthquakeStructTags(t *testing.T) {
	e := Earthquake{}
	if e.Date != "" {
		t.Error("expected empty Earthquake")
	}
}

func TestAutoGempaResponseStruct(t *testing.T) {
	var resp AutoGempaResponse
	if resp.Infogempa.Gempa.Date != "" {
		t.Error("expected empty AutoGempaResponse")
	}
}

func TestGempaListResponseStruct(t *testing.T) {
	var resp GempaListResponse
	if resp.Infogempa.Gempa != nil {
		t.Error("expected nil Gempa slice")
	}
}

func TestParsedEarthquakeDefaults(t *testing.T) {
	p := ParsedEarthquake{}
	if p.Magnitude != 0 {
		t.Errorf("expected zero magnitude, got %f", p.Magnitude)
	}
	if p.DepthKM != 0 {
		t.Errorf("expected zero depth, got %f", p.DepthKM)
	}
	if len(p.Coordinates) != 0 {
		t.Errorf("expected empty coordinates, got %v", p.Coordinates)
	}
}
