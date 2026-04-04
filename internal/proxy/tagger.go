package proxy

import "net/http"

const (
	headerAgentID  = "X-Meowsight-Agent"
	headerTenantID = "X-Meowsight-Tenant"
)

// TagFromRequest extracts agent and tenant identification from the request.
// Priority: custom headers > API key prefix parsing (future).
func TagFromRequest(r *http.Request) (tenantID, agentID string) {
	tenantID = r.Header.Get(headerTenantID)
	agentID = r.Header.Get(headerAgentID)

	if tenantID == "" {
		tenantID = "default"
	}
	if agentID == "" {
		agentID = "unknown"
	}
	return tenantID, agentID
}
