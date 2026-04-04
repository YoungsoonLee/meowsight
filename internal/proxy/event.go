package proxy

import "time"

// RequestEvent captures data from a proxied LLM request for downstream processing.
type RequestEvent struct {
	TenantID     string
	AgentID      string
	Provider     string
	Model        string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	LatencyMs    int64
	StatusCode   int
	Error        string
	Streaming    bool
	Timestamp    time.Time
}

// EventEmitter sends proxy events to the pipeline.
// Implementations: NATS (production), logger (development).
type EventEmitter interface {
	Emit(event RequestEvent)
}
