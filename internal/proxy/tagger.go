package proxy

import "net/http"

const (
	headerAgentID  = "X-Meowsight-Agent"
	headerTenantID = "X-Meowsight-Tenant"
)

// TagResult holds the resolved identity for a request.
type TagResult struct {
	TenantID       string
	AgentID        string
	UpstreamAPIKey string // non-empty if resolved via API key (needs key swap)
}

// TagFromRequest extracts agent and tenant identification from the request.
// Priority: 1) X-Meowsight-* headers, 2) API key resolver, 3) defaults.
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

// TagFromRequestWithKey extracts identity using headers first, then falls back
// to API key resolution. Returns a TagResult with optional upstream API key for swapping.
func TagFromRequestWithKey(r *http.Request, resolver *KeyResolver) TagResult {
	// Priority 1: explicit headers
	tenantID := r.Header.Get(headerTenantID)
	agentID := r.Header.Get(headerAgentID)

	if tenantID != "" && agentID != "" {
		return TagResult{TenantID: tenantID, AgentID: agentID}
	}

	// Priority 2: API key resolution
	if resolver != nil {
		apiKey := extractAPIKey(r)
		if apiKey != "" {
			if mapping := resolver.Resolve(r.Context(), apiKey); mapping != nil {
				result := TagResult{
					TenantID:       mapping.TenantID,
					AgentID:        mapping.AgentID,
					UpstreamAPIKey: mapping.UpstreamAPIKey,
				}
				// Fill in any missing fields from headers
				if tenantID != "" {
					result.TenantID = tenantID
				}
				if agentID != "" {
					result.AgentID = agentID
				}
				return result
			}
		}
	}

	// Priority 3: defaults
	if tenantID == "" {
		tenantID = "default"
	}
	if agentID == "" {
		agentID = "unknown"
	}
	return TagResult{TenantID: tenantID, AgentID: agentID}
}

// extractAPIKey gets the API key from the request headers.
// Supports: Authorization: Bearer <key> (OpenAI) and x-api-key: <key> (Anthropic).
func extractAPIKey(r *http.Request) string {
	// OpenAI style: Authorization: Bearer sk-...
	if auth := r.Header.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	// Anthropic style: x-api-key: sk-ant-...
	if key := r.Header.Get("x-api-key"); key != "" {
		return key
	}
	return ""
}
