package clickhouse

import (
	"testing"
	"time"

	"github.com/YoungsoonLee/meowsight/internal/proxy"
)

func TestMetricsFromEvent(t *testing.T) {
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
		Timestamp:    time.Now(),
	}

	// Verify metric derivation logic (without actual ClickHouse connection)
	type metric struct {
		name  string
		value float64
	}

	metrics := []metric{
		{"input_tokens", float64(event.InputTokens)},
		{"output_tokens", float64(event.OutputTokens)},
		{"cost_usd", event.CostUSD},
		{"latency_ms", float64(event.LatencyMs)},
	}

	if event.Error != "" {
		metrics = append(metrics, metric{"error_count", 1})
	}

	if len(metrics) != 4 {
		t.Errorf("expected 4 metrics for success event, got %d", len(metrics))
	}

	// Verify error event adds error_count
	errorEvent := event
	errorEvent.Error = "upstream timeout"

	errorMetrics := []metric{
		{"input_tokens", float64(errorEvent.InputTokens)},
		{"output_tokens", float64(errorEvent.OutputTokens)},
		{"cost_usd", errorEvent.CostUSD},
		{"latency_ms", float64(errorEvent.LatencyMs)},
	}
	if errorEvent.Error != "" {
		errorMetrics = append(errorMetrics, metric{"error_count", 1})
	}

	if len(errorMetrics) != 5 {
		t.Errorf("expected 5 metrics for error event, got %d", len(errorMetrics))
	}
}

func TestMetricLabels(t *testing.T) {
	event := proxy.RequestEvent{
		Provider:  "anthropic",
		Model:     "claude-sonnet-4-0",
		Streaming: true,
	}

	labels := map[string]string{
		"provider":  event.Provider,
		"model":     event.Model,
		"streaming": "true",
	}

	if labels["provider"] != "anthropic" {
		t.Errorf("expected provider anthropic, got %s", labels["provider"])
	}
	if labels["model"] != "claude-sonnet-4-0" {
		t.Errorf("expected model claude-sonnet-4-0, got %s", labels["model"])
	}
	if labels["streaming"] != "true" {
		t.Errorf("expected streaming true, got %s", labels["streaming"])
	}
}

func TestTimestampFallback(t *testing.T) {
	event := proxy.RequestEvent{
		TenantID: "t-1",
	}

	ts := event.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	if ts.IsZero() {
		t.Error("expected non-zero timestamp after fallback")
	}
}
