package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	pgadapter "github.com/YoungsoonLee/meowsight/internal/adapter/postgres"
)

// TenantHandler provides REST endpoints for tenant management.
type TenantHandler struct {
	repo *pgadapter.TenantRepo
}

// NewTenantHandler creates a new TenantHandler.
func NewTenantHandler(repo *pgadapter.TenantRepo) *TenantHandler {
	return &TenantHandler{repo: repo}
}

// RegisterRoutes mounts tenant endpoints on the given mux.
func (h *TenantHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/tenants", h.Create)
	mux.HandleFunc("GET /api/v1/tenants", h.List)
	mux.HandleFunc("GET /api/v1/tenants/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/tenants/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/tenants/{id}", h.Delete)
	mux.HandleFunc("POST /api/v1/tenants/{id}/rotate-key", h.RotateKey)
}

// Create handles POST /api/v1/tenants
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Plan string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	tenant, err := h.repo.Create(r.Context(), req.Name, req.Plan)
	if err != nil {
		slog.Error("create tenant failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create tenant")
		return
	}

	writeJSON(w, http.StatusCreated, tenant)
}

// List handles GET /api/v1/tenants
func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.repo.List(r.Context())
	if err != nil {
		slog.Error("list tenants failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list tenants")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tenants": tenants,
		"total":   len(tenants),
	})
}

// Get handles GET /api/v1/tenants/{id}
func (h *TenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	tenant, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("get tenant failed", "error", err, "id", id)
		writeError(w, http.StatusInternalServerError, "failed to get tenant")
		return
	}
	if tenant == nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	writeJSON(w, http.StatusOK, tenant)
}

// Update handles PUT /api/v1/tenants/{id}
func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Name string `json:"name"`
		Plan string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Plan == "" {
		req.Plan = "free"
	}

	tenant, err := h.repo.Update(r.Context(), id, req.Name, req.Plan)
	if err != nil {
		slog.Error("update tenant failed", "error", err, "id", id)
		writeError(w, http.StatusInternalServerError, "failed to update tenant")
		return
	}
	if tenant == nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	writeJSON(w, http.StatusOK, tenant)
}

// Delete handles DELETE /api/v1/tenants/{id}
func (h *TenantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.repo.Delete(r.Context(), id); err != nil {
		slog.Error("delete tenant failed", "error", err, "id", id)
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// RotateKey handles POST /api/v1/tenants/{id}/rotate-key
func (h *TenantHandler) RotateKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	newKey, err := h.repo.RotateAPIKey(r.Context(), id)
	if err != nil {
		slog.Error("rotate key failed", "error", err, "id", id)
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"api_key": newKey,
		"message": "Store this key securely. It will not be shown again.",
	})
}
