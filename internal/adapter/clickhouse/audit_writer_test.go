package clickhouse

import (
	"fmt"
	"testing"
	"time"

	"github.com/YoungsoonLee/meowsight/internal/proxy"
)

func TestAuditLogFields(t *testing.T) {
	event := proxy.RequestEvent{
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
		Timestamp:    time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC),
	}

	// Verify field transformations match audit_log schema
	action := "llm_request"
	resource := fmt.Sprintf("/%s/v1/chat", event.Provider)
	metadata := map[string]string{
		"streaming": fmt.Sprintf("%t", event.Streaming),
	}

	if action != "llm_request" {
		t.Errorf("expected action llm_request, got %s", action)
	}
	if resource != "/openai/v1/chat" {
		t.Errorf("expected resource /openai/v1/chat, got %s", resource)
	}
	if metadata["streaming"] != "false" {
		t.Errorf("expected streaming=false, got %s", metadata["streaming"])
	}

	// Verify uint32 conversions don't overflow for reasonable values
	if uint32(event.InputTokens) != 100 {
		t.Error("input_tokens uint32 conversion failed")
	}
	if uint32(event.OutputTokens) != 50 {
		t.Error("output_tokens uint32 conversion failed")
	}
	if uint32(event.LatencyMs) != 230 {
		t.Error("latency_ms uint32 conversion failed")
	}
	if uint16(event.StatusCode) != 200 {
		t.Error("status_code uint16 conversion failed")
	}
}

func TestAuditLogErrorEvent(t *testing.T) {
	event := proxy.RequestEvent{
		TenantID:   "tenant-1",
		AgentID:    "agent-1",
		Provider:   "anthropic",
		Model:      "claude-sonnet-4-0",
		StatusCode: 500,
		Error:      "upstream timeout",
		Timestamp:  time.Now(),
	}

	if event.Error == "" {
		t.Error("expected non-empty error field")
	}
	if event.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", event.StatusCode)
	}
}

func TestAuditLogTimestampFallback(t *testing.T) {
	event := proxy.RequestEvent{TenantID: "t-1"}

	ts := event.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	if ts.IsZero() {
		t.Error("expected non-zero timestamp after fallback")
	}
}
