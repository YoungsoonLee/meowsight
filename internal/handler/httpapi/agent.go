package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	pgadapter "github.com/YoungsoonLee/meowsight/internal/adapter/postgres"
)

// AgentHandler provides REST endpoints for agent management.
type AgentHandler struct {
	repo *pgadapter.AgentRepo
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(repo *pgadapter.AgentRepo) *AgentHandler {
	return &AgentHandler{repo: repo}
}

// RegisterRoutes mounts agent management endpoints on the given mux.
func (h *AgentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/agents/all", h.ListAll)
	mux.HandleFunc("GET /api/v1/agents/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/agents/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/agents/{id}", h.Delete)
}

// ListAll handles GET /api/v1/agents/all — returns all agents across tenants.
func (h *AgentHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	agents, err := h.repo.ListAll(r.Context())
	if err != nil {
		slog.Error("list all agents failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list agents")
		return
	}

	cutoff := time.Now().Add(-10 * time.Minute)
	type resp struct {
		ID           string    `json:"id"`
		TenantUUID   *string   `json:"tenant_uuid,omitempty"`
		TenantID     string    `json:"tenant_id"`
		AgentID      string    `json:"agent_id"`
		Name         string    `json:"name"`
		Status       string    `json:"status"`
		Provider     string    `json:"provider"`
		Model        string    `json:"model"`
		FirstSeenAt  time.Time `json:"first_seen_at"`
		LastSeenAt   time.Time `json:"last_seen_at"`
		RequestCount int64     `json:"request_count"`
		Active       bool      `json:"active"`
		Linked       bool      `json:"linked"`
	}

	result := make([]resp, 0, len(agents))
	for _, a := range agents {
		result = append(result, resp{
			ID:           a.ID,
			TenantUUID:   a.TenantUUID,
			TenantID:     a.ExternalTenantID,
			AgentID:      a.ExternalAgentID,
			Name:         a.Name,
			Status:       a.Status,
			Provider:     a.Provider,
			Model:        a.Model,
			FirstSeenAt:  a.FirstSeenAt,
			LastSeenAt:   a.LastSeenAt,
			RequestCount: a.RequestCount,
			Active:       a.LastSeenAt.After(cutoff),
			Linked:       a.TenantUUID != nil,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"agents": result,
		"total":  len(result),
	})
}

// Get handles GET /api/v1/agents/{id}
func (h *AgentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	agent, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("get agent failed", "error", err, "id", id)
		writeError(w, http.StatusInternalServerError, "failed to get agent")
		return
	}
	if agent == nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	writeJSON(w, http.StatusOK, agent)
}

// Update handles PUT /api/v1/agents/{id}
func (h *AgentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Name     string  `json:"name"`
		Status   string  `json:"status"`
		TenantID *string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}

	if err := h.repo.Update(r.Context(), id, req.Name, req.Status, req.TenantID); err != nil {
		slog.Error("update agent failed", "error", err, "id", id)
		writeError(w, http.StatusInternalServerError, "failed to update agent")
		return
	}

	agent, err := h.repo.GetByID(r.Context(), id)
	if err != nil || agent == nil {
		writeError(w, http.StatusNotFound, "agent not found after update")
		return
	}

	writeJSON(w, http.StatusOK, agent)
}

// Delete handles DELETE /api/v1/agents/{id}
func (h *AgentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.repo.Delete(r.Context(), id); err != nil {
		slog.Error("delete agent failed", "error", err, "id", id)
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
