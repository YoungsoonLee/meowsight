package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/YoungsoonLee/meowsight/internal/proxy"
)

// testEmitter collects emitted events for assertions.
type testEmitter struct {
	mu     sync.Mutex
	events []proxy.RequestEvent
}

func (e *testEmitter) Emit(ev proxy.RequestEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, ev)
}

func (e *testEmitter) last() proxy.RequestEvent {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.events[len(e.events)-1]
}

func TestOpenAI_NonStreaming(t *testing.T) {
	// Mock OpenAI API
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"model": "gpt-4o",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "Hello!"}},
			},
			"usage": map[string]int{
				"prompt_tokens":     100,
				"completion_tokens": 50,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer upstream.Close()

	emitter := &testEmitter{}
	oai := &OpenAI{name: "openai", prefix: "/openai", target: mustParseURL(upstream.URL), pricing: testPricing(), emitter: emitter}

	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/openai/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("X-Meowsight-Tenant", "tenant-1")
	req.Header.Set("X-Meowsight-Agent", "agent-1")

	rec := httptest.NewRecorder()
	oai.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify response body is passed through
	var result map[string]any
	json.Unmarshal(rec.Body.Bytes(), &result)
	if result["model"] != "gpt-4o" {
		t.Errorf("expected model gpt-4o in response, got %v", result["model"])
	}

	// Verify event was emitted
	if len(emitter.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(emitter.events))
	}
	ev := emitter.last()
	if ev.Provider != "openai" {
		t.Errorf("expected provider openai, got %s", ev.Provider)
	}
	if ev.Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", ev.Model)
	}
	if ev.InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", ev.InputTokens)
	}
	if ev.OutputTokens != 50 {
		t.Errorf("expected 50 output tokens, got %d", ev.OutputTokens)
	}
	if ev.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", ev.TenantID)
	}
	if ev.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", ev.AgentID)
	}
	if ev.CostUSD <= 0 {
		t.Error("expected positive cost")
	}
	if ev.Streaming {
		t.Error("expected non-streaming")
	}
}

func TestOpenAI_Streaming(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify stream_options was injected
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		so, ok := body["stream_options"].(map[string]any)
		if !ok || so["include_usage"] != true {
			t.Error("expected stream_options.include_usage=true to be injected")
		}

		flusher, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`data: {"model":"gpt-4o","choices":[{"delta":{"content":"Hi"}}]}`,
			`data: {"model":"gpt-4o","choices":[{"delta":{"content":"!"}}]}`,
			`data: {"model":"gpt-4o","choices":[],"usage":{"prompt_tokens":80,"completion_tokens":20}}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			fmt.Fprintln(w, c)
			flusher.Flush()
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer upstream.Close()

	emitter := &testEmitter{}
	oai := &OpenAI{name: "openai", prefix: "/openai", target: mustParseURL(upstream.URL), pricing: testPricing(), emitter: emitter}

	body := `{"model":"gpt-4o","stream":true,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/openai/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("X-Meowsight-Tenant", "t-1")
	req.Header.Set("X-Meowsight-Agent", "a-1")

	rec := httptest.NewRecorder()
	oai.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Verify SSE content passed through
	respBody := rec.Body.String()
	if !strings.Contains(respBody, "Hi") {
		t.Error("expected streamed content in response")
	}

	if len(emitter.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(emitter.events))
	}
	ev := emitter.last()
	if ev.InputTokens != 80 {
		t.Errorf("expected 80 input tokens, got %d", ev.InputTokens)
	}
	if ev.OutputTokens != 20 {
		t.Errorf("expected 20 output tokens, got %d", ev.OutputTokens)
	}
	if !ev.Streaming {
		t.Error("expected streaming=true")
	}
}

func testPricing() *proxy.PricingTable {
	pt := proxy.NewPricingTable()
	pt.LoadFromFile("../../../configs/pricing.json")
	return pt
}

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}
