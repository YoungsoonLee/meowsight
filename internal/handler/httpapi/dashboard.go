package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	chadapter "github.com/YoungsoonLee/meowsight/internal/adapter/clickhouse"
	pgadapter "github.com/YoungsoonLee/meowsight/internal/adapter/postgres"
)

// DashboardHandler provides REST endpoints for dashboard queries.
type DashboardHandler struct {
	agentRepo    *pgadapter.AgentRepo
	metricReader *chadapter.MetricReader
}

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(agentRepo *pgadapter.AgentRepo, metricReader *chadapter.MetricReader) *DashboardHandler {
	return &DashboardHandler{
		agentRepo:    agentRepo,
		metricReader: metricReader,
	}
}

// RegisterRoutes mounts all dashboard endpoints on the given mux.
func (h *DashboardHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/agents", h.ListAgents)
	mux.HandleFunc("GET /api/v1/metrics/summary", h.MetricsSummary)
	mux.HandleFunc("GET /api/v1/audit", h.AuditLogs)
}

// ListAgents returns all discovered agents for a tenant.
// GET /api/v1/agents?tenant_id=xxx
func (h *DashboardHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	agents, err := h.agentRepo.GetByTenant(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query agents")
		return
	}

	// Determine active status based on last_seen_at
	type agentResponse struct {
		ID           string    `json:"id"`
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
	}

	cutoff := time.Now().Add(-10 * time.Minute)
	result := make([]agentResponse, 0, len(agents))
	for _, a := range agents {
		result = append(result, agentResponse{
			ID:           a.ID,
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
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"agents":    result,
		"total":     len(result),
	})
}

// MetricsSummary returns aggregated metrics for a tenant.
// GET /api/v1/metrics/summary?tenant_id=xxx&from=2026-04-01T00:00:00Z&to=2026-04-05T23:59:59Z
func (h *DashboardHandler) MetricsSummary(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	from, to := parseTimeRange(r)

	summary, err := h.metricReader.GetSummary(r.Context(), tenantID, from, to)
	if err != nil {
		slog.Error("metrics summary query failed", "error", err, "tenant_id", tenantID)
		writeError(w, http.StatusInternalServerError, "failed to query metrics")
		return
	}

	// Calculate totals
	var totalCost float64
	var totalInput, totalOutput float64
	for _, s := range summary {
		totalCost += s.TotalCost
		totalInput += s.TotalInput
		totalOutput += s.TotalOutput
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id":          tenantID,
		"from":               from,
		"to":                 to,
		"breakdown":          summary,
		"total_cost_usd":     totalCost,
		"total_input_tokens": totalInput,
		"total_output_tokens": totalOutput,
	})
}

// AuditLogs returns recent audit log entries for a tenant.
// GET /api/v1/audit?tenant_id=xxx&limit=50&offset=0
func (h *DashboardHandler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	logs, err := h.metricReader.GetAuditLogs(r.Context(), tenantID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query audit logs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": tenantID,
		"logs":      logs,
		"limit":     limit,
		"offset":    offset,
	})
}

// parseTimeRange extracts from/to query params with sensible defaults (last 24h).
func parseTimeRange(r *http.Request) (time.Time, time.Time) {
	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	to := now

	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		}
	}
	return from, to
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
