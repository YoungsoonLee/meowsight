package nats

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/YoungsoonLee/meowsight/internal/proxy"
)

func TestEventMarshal(t *testing.T) {
	ev := proxy.RequestEvent{
		TenantID:     "tenant-1",
		AgentID:      "agent-1",
		Provider:     "openai",
		Model:        "gpt-4o",
		InputTokens:  100,
		OutputTokens: 50,
		CostUSD:      0.0075,
		LatencyMs:    230,
		StatusCode:   200,
		Streaming:    false,
		Timestamp:    time.Now(),
	}

	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var decoded proxy.RequestEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if decoded.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", decoded.TenantID)
	}
	if decoded.Model != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", decoded.Model)
	}
	if decoded.InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", decoded.InputTokens)
	}
	if decoded.CostUSD != 0.0075 {
		t.Errorf("expected cost 0.0075, got %f", decoded.CostUSD)
	}
}

func TestSubjectFormat(t *testing.T) {
	tests := []struct {
		tenantID string
		want     string
	}{
		{"tenant-1", "events.tenant-1.request"},
		{"", "events.default.request"},
	}

	for _, tt := range tests {
		tid := tt.tenantID
		if tid == "" {
			tid = "default"
		}
		subject := subjectPrefix + "." + tid + ".request"
		if subject != tt.want {
			t.Errorf("expected subject %s, got %s", tt.want, subject)
		}
	}
}
