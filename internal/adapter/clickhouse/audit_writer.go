package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"

	"github.com/YoungsoonLee/meowsight/internal/proxy"
)

// AuditWriter writes proxy RequestEvents as audit log entries to ClickHouse.
type AuditWriter struct {
	conn driver.Conn
}

// NewAuditWriter connects to ClickHouse and returns an audit writer.
func NewAuditWriter(ctx context.Context, host string, port int, database, user, password string) (*AuditWriter, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", host, port)},
		Auth: clickhouse.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:     10 * time.Second,
		ConnMaxLifetime: 1 * time.Hour,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}

	slog.Info("clickhouse audit writer ready", "host", host, "port", port, "database", database)
	return &AuditWriter{conn: conn}, nil
}

// WriteAuditLog inserts an audit log entry from a RequestEvent.
func (w *AuditWriter) WriteAuditLog(ctx context.Context, event proxy.RequestEvent) error {
	ts := event.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	metadata := map[string]string{
		"streaming": fmt.Sprintf("%t", event.Streaming),
	}

	err := w.conn.Exec(ctx,
		`INSERT INTO audit_log (id, tenant_id, agent_id, action, resource, provider, model,
		input_tokens, output_tokens, cost_usd, latency_ms, status_code, error, metadata, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(),
		event.TenantID,
		event.AgentID,
		"llm_request",
		fmt.Sprintf("/%s/v1/chat", event.Provider),
		event.Provider,
		event.Model,
		uint32(event.InputTokens),
		uint32(event.OutputTokens),
		event.CostUSD,
		uint32(event.LatencyMs),
		uint16(event.StatusCode),
		event.Error,
		metadata,
		ts,
	)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return nil
}

// Close closes the ClickHouse connection.
func (w *AuditWriter) Close() error {
	return w.conn.Close()
}
