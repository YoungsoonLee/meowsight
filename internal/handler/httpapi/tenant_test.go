package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTenantCreate_MissingName(t *testing.T) {
	h := &TenantHandler{}
	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"plan":"free"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", body)
	req.Header.Set("Content-Type", "application/json")

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "name is required") {
		t.Errorf("expected 'name is required' in body, got %s", rec.Body.String())
	}
}

func TestTenantCreate_InvalidJSON(t *testing.T) {
	h := &TenantHandler{}
	rec := httptest.NewRecorder()
	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", body)

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestTenantUpdate_MissingName(t *testing.T) {
	h := &TenantHandler{}
	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"plan":"pro"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/some-id", body)
	req.Header.Set("Content-Type", "application/json")

	h.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
