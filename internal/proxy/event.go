package proxy

import "time"

// RequestEvent captures data from a proxied LLM request for downstream processing.
type RequestEvent struct {
	TenantID     string    `json:"tenant_id"`
	AgentID      string    `json:"agent_id"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	LatencyMs    int64     `json:"latency_ms"`
	StatusCode   int       `json:"status_code"`
	Error        string    `json:"error,omitempty"`
	Streaming    bool      `json:"streaming"`
	Timestamp    time.Time `json:"timestamp"`
}

// EventEmitter sends proxy events to the pipeline.
// Implementations: NATS (production), logger (development).
type EventEmitter interface {
	Emit(event RequestEvent)
}
