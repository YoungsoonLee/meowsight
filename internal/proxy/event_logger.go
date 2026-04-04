package proxy

import "log/slog"

// LogEmitter logs events via slog. Used during development before NATS is wired up.
type LogEmitter struct{}

func (l *LogEmitter) Emit(e RequestEvent) {
	slog.Info("proxy.event",
		"tenant_id", e.TenantID,
		"agent_id", e.AgentID,
		"provider", e.Provider,
		"model", e.Model,
		"input_tokens", e.InputTokens,
		"output_tokens", e.OutputTokens,
		"cost_usd", e.CostUSD,
		"latency_ms", e.LatencyMs,
		"status_code", e.StatusCode,
		"streaming", e.Streaming,
		"error", e.Error,
	)
}
