package proxy

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTagFromRequest(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		wantTenant string
		wantAgent  string
	}{
		{
			name:       "with headers",
			headers:    map[string]string{"X-Meowsight-Tenant": "t-123", "X-Meowsight-Agent": "agent-1"},
			wantTenant: "t-123",
			wantAgent:  "agent-1",
		},
		{
			name:       "no headers defaults",
			headers:    map[string]string{},
			wantTenant: "default",
			wantAgent:  "unknown",
		},
		{
			name:       "partial headers",
			headers:    map[string]string{"X-Meowsight-Agent": "my-agent"},
			wantTenant: "default",
			wantAgent:  "my-agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/openai/v1/chat/completions", nil)
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			tenant, agent := TagFromRequest(r)
			if tenant != tt.wantTenant {
				t.Errorf("tenant: got %q, want %q", tenant, tt.wantTenant)
			}
			if agent != tt.wantAgent {
				t.Errorf("agent: got %q, want %q", agent, tt.wantAgent)
			}
		})
	}
}

func TestTagFromRequestWithKey(t *testing.T) {
	store := &testKeyStore{
		keys: map[string]*KeyMapping{
			HashKey("ms-cursor-key-123"): {
				TenantID:       "acme-corp",
				AgentID:        "cursor-dev",
				Provider:       "openai",
				UpstreamAPIKey: "sk-real-key",
			},
		},
	}
	resolver := NewKeyResolver(store, 5*time.Minute)

	tests := []struct {
		name           string
		headers        map[string]string
		wantTenant     string
		wantAgent      string
		wantUpstreamKey string
	}{
		{
			name:       "headers take priority",
			headers:    map[string]string{"X-Meowsight-Tenant": "t-1", "X-Meowsight-Agent": "a-1", "Authorization": "Bearer ms-cursor-key-123"},
			wantTenant: "t-1",
			wantAgent:  "a-1",
		},
		{
			name:           "fallback to API key",
			headers:        map[string]string{"Authorization": "Bearer ms-cursor-key-123"},
			wantTenant:     "acme-corp",
			wantAgent:      "cursor-dev",
			wantUpstreamKey: "sk-real-key",
		},
		{
			name:       "unknown API key falls to defaults",
			headers:    map[string]string{"Authorization": "Bearer unknown-key"},
			wantTenant: "default",
			wantAgent:  "unknown",
		},
		{
			name:       "no headers no key",
			headers:    map[string]string{},
			wantTenant: "default",
			wantAgent:  "unknown",
		},
		{
			name:           "anthropic x-api-key",
			headers:        map[string]string{"x-api-key": "ms-cursor-key-123"},
			wantTenant:     "acme-corp",
			wantAgent:      "cursor-dev",
			wantUpstreamKey: "sk-real-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/openai/v1/chat/completions", nil)
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			result := TagFromRequestWithKey(r, resolver)
			if result.TenantID != tt.wantTenant {
				t.Errorf("tenant: got %q, want %q", result.TenantID, tt.wantTenant)
			}
			if result.AgentID != tt.wantAgent {
				t.Errorf("agent: got %q, want %q", result.AgentID, tt.wantAgent)
			}
			if result.UpstreamAPIKey != tt.wantUpstreamKey {
				t.Errorf("upstream key: got %q, want %q", result.UpstreamAPIKey, tt.wantUpstreamKey)
			}
		})
	}
}

// testKeyStore implements KeyStore for tagger tests.
type testKeyStore struct {
	keys map[string]*KeyMapping
}

func (s *testKeyStore) LookupByHash(_ context.Context, keyHash string) (*KeyMapping, error) {
	return s.keys[keyHash], nil
}
