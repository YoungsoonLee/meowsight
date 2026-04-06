package httpapi

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseTimeRange_Defaults(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/metrics/summary", nil)
	from, to := parseTimeRange(r)

	if to.Sub(from) < 23*time.Hour {
		t.Errorf("expected ~24h range, got %v", to.Sub(from))
	}
}

func TestParseTimeRange_Custom(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/metrics/summary?from=2026-04-01T00:00:00Z&to=2026-04-05T23:59:59Z", nil)
	from, to := parseTimeRange(r)

	expectedFrom := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	expectedTo := time.Date(2026, 4, 5, 23, 59, 59, 0, time.UTC)

	if !from.Equal(expectedFrom) {
		t.Errorf("expected from %v, got %v", expectedFrom, from)
	}
	if !to.Equal(expectedTo) {
		t.Errorf("expected to %v, got %v", expectedTo, to)
	}
}

func TestParseTimeRange_InvalidFallback(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/metrics/summary?from=bad&to=bad", nil)
	from, to := parseTimeRange(r)

	// Should fall back to defaults
	if to.Sub(from) < 23*time.Hour {
		t.Errorf("expected ~24h default range on invalid input, got %v", to.Sub(from))
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, 200, map[string]string{"status": "ok"})

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, 400, "bad request")

	if rec.Code != 400 {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body == "" {
		t.Error("expected non-empty error body")
	}
}
