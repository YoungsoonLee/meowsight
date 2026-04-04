package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAnthropic_NonStreaming(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"model": "claude-sonnet-4-0",
			"content": []map[string]string{
				{"type": "text", "text": "Hello!"},
			},
			"usage": map[string]int{
				"input_tokens":  200,
				"output_tokens": 100,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer upstream.Close()

	emitter := &testEmitter{}
	ant := &Anthropic{name: "anthropic", prefix: "/anthropic", target: mustParseURL(upstream.URL), pricing: testPricing(), emitter: emitter}

	body := `{"model":"claude-sonnet-4-0","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", "sk-ant-test")
	req.Header.Set("X-Meowsight-Tenant", "tenant-2")
	req.Header.Set("X-Meowsight-Agent", "agent-2")

	rec := httptest.NewRecorder()
	ant.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if len(emitter.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(emitter.events))
	}
	ev := emitter.last()
	if ev.Provider != "anthropic" {
		t.Errorf("expected provider anthropic, got %s", ev.Provider)
	}
	if ev.Model != "claude-sonnet-4-0" {
		t.Errorf("expected model claude-sonnet-4-0, got %s", ev.Model)
	}
	if ev.InputTokens != 200 {
		t.Errorf("expected 200 input tokens, got %d", ev.InputTokens)
	}
	if ev.OutputTokens != 100 {
		t.Errorf("expected 100 output tokens, got %d", ev.OutputTokens)
	}
	if ev.TenantID != "tenant-2" {
		t.Errorf("expected tenant-2, got %s", ev.TenantID)
	}
}

func TestAnthropic_Streaming(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		events := []string{
			"event: message_start\ndata: {\"message\":{\"model\":\"claude-sonnet-4-0\",\"usage\":{\"input_tokens\":150}}}",
			"event: content_block_delta\ndata: {\"delta\":{\"text\":\"Hi\"}}",
			"event: message_delta\ndata: {\"usage\":{\"output_tokens\":75}}",
			"event: message_stop\ndata: {}",
		}
		for _, e := range events {
			fmt.Fprintln(w, e)
			flusher.Flush()
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer upstream.Close()

	emitter := &testEmitter{}
	ant := &Anthropic{name: "anthropic", prefix: "/anthropic", target: mustParseURL(upstream.URL), pricing: testPricing(), emitter: emitter}

	body := `{"model":"claude-sonnet-4-0","stream":true,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", "sk-ant-test")

	rec := httptest.NewRecorder()
	ant.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if len(emitter.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(emitter.events))
	}
	ev := emitter.last()
	if ev.InputTokens != 150 {
		t.Errorf("expected 150 input tokens, got %d", ev.InputTokens)
	}
	if ev.OutputTokens != 75 {
		t.Errorf("expected 75 output tokens, got %d", ev.OutputTokens)
	}
	if !ev.Streaming {
		t.Error("expected streaming=true")
	}
	if ev.Model != "claude-sonnet-4-0" {
		t.Errorf("expected model claude-sonnet-4-0, got %s", ev.Model)
	}
}
