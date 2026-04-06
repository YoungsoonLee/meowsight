package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// MetricSummary holds aggregated metric data for dashboard display.
type MetricSummary struct {
	TenantID     string  `json:"tenant_id"`
	AgentID      string  `json:"agent_id"`
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	TotalInput   float64 `json:"total_input_tokens"`
	TotalOutput  float64 `json:"total_output_tokens"`
	TotalCost    float64 `json:"total_cost_usd"`
	AvgLatency   float64 `json:"avg_latency_ms"`
	ErrorCount   float64 `json:"error_count"`
	RequestCount uint64  `json:"request_count"`
}

// AuditEntry holds a single audit log record.
type AuditEntry struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id"`
	AgentID      string            `json:"agent_id"`
	Action       string            `json:"action"`
	Provider     string            `json:"provider"`
	Model        string            `json:"model"`
	InputTokens  uint32            `json:"input_tokens"`
	OutputTokens uint32            `json:"output_tokens"`
	CostUSD      float64           `json:"cost_usd"`
	LatencyMs    uint32            `json:"latency_ms"`
	StatusCode   uint16            `json:"status_code"`
	Error        string            `json:"error,omitempty"`
	Metadata     map[string]string `json:"metadata"`
	Timestamp    time.Time         `json:"timestamp"`
}

// MetricReader reads metrics and audit logs from ClickHouse for dashboard queries.
type MetricReader struct {
	conn driver.Conn
}

// NewMetricReader connects to ClickHouse and returns a reader.
func NewMetricReader(ctx context.Context, host string, port int, database, user, password string) (*MetricReader, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", host, port)},
		Auth: clickhouse.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
		DialTimeout:     10 * time.Second,
		ConnMaxLifetime: 1 * time.Hour,
		MaxOpenConns:    5,
		MaxIdleConns:    3,
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}

	return &MetricReader{conn: conn}, nil
}

// GetSummary returns aggregated metrics per agent for a tenant within a time range.
func (r *MetricReader) GetSummary(ctx context.Context, tenantID string, from, to time.Time) ([]MetricSummary, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT
			tenant_id,
			agent_id,
			labels['provider'] AS provider,
			labels['model'] AS model,
			sumIf(value, metric_name = 'input_tokens') AS total_input,
			sumIf(value, metric_name = 'output_tokens') AS total_output,
			sumIf(value, metric_name = 'cost_usd') AS total_cost,
			avgIf(value, metric_name = 'latency_ms') AS avg_latency,
			sumIf(value, metric_name = 'error_count') AS error_count,
			count() / 4 AS request_count
		FROM metrics
		WHERE tenant_id = $1 AND timestamp >= $2 AND timestamp <= $3
		GROUP BY tenant_id, agent_id, provider, model
		ORDER BY total_cost DESC
	`, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query summary: %w", err)
	}
	defer rows.Close()

	var results []MetricSummary
	for rows.Next() {
		var s MetricSummary
		if err := rows.Scan(&s.TenantID, &s.AgentID, &s.Provider, &s.Model,
			&s.TotalInput, &s.TotalOutput, &s.TotalCost, &s.AvgLatency,
			&s.ErrorCount, &s.RequestCount); err != nil {
			return nil, fmt.Errorf("scan summary: %w", err)
		}
		results = append(results, s)
	}
	return results, nil
}

// GetAuditLogs returns recent audit log entries for a tenant.
func (r *MetricReader) GetAuditLogs(ctx context.Context, tenantID string, limit int, offset int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := r.conn.Query(ctx, `
		SELECT id, tenant_id, agent_id, action, provider, model,
		       input_tokens, output_tokens, cost_usd, latency_ms,
		       status_code, error, metadata, timestamp
		FROM audit_log
		WHERE tenant_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var results []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.TenantID, &e.AgentID, &e.Action, &e.Provider, &e.Model,
			&e.InputTokens, &e.OutputTokens, &e.CostUSD, &e.LatencyMs,
			&e.StatusCode, &e.Error, &e.Metadata, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		results = append(results, e)
	}
	return results, nil
}

// Close closes the ClickHouse connection.
func (r *MetricReader) Close() error {
	return r.conn.Close()
}
