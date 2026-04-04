package proxy

import (
	"net/http/httptest"
	"testing"
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
