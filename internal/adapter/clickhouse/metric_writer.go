package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/YoungsoonLee/meowsight/internal/proxy"
)

// MetricWriter writes proxy RequestEvents as metrics to ClickHouse.
type MetricWriter struct {
	conn driver.Conn
}

// NewMetricWriter connects to ClickHouse and returns a writer.
func NewMetricWriter(ctx context.Context, host string, port int, database, user, password string) (*MetricWriter, error) {
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

	slog.Info("clickhouse metric writer ready", "host", host, "port", port, "database", database)
	return &MetricWriter{conn: conn}, nil
}

// WriteMetrics inserts metric rows derived from a RequestEvent.
// Each event produces multiple metrics: input_tokens, output_tokens, cost_usd, latency_ms, error_count.
func (w *MetricWriter) WriteMetrics(ctx context.Context, event proxy.RequestEvent) error {
	labels := map[string]string{
		"provider":  event.Provider,
		"model":     event.Model,
		"streaming": fmt.Sprintf("%t", event.Streaming),
	}

	metrics := []struct {
		name  string
		value float64
	}{
		{"input_tokens", float64(event.InputTokens)},
		{"output_tokens", float64(event.OutputTokens)},
		{"cost_usd", event.CostUSD},
		{"latency_ms", float64(event.LatencyMs)},
	}

	if event.Error != "" {
		metrics = append(metrics, struct {
			name  string
			value float64
		}{"error_count", 1})
	}

	batch, err := w.conn.PrepareBatch(ctx, "INSERT INTO metrics (tenant_id, agent_id, metric_name, value, labels, timestamp)")
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	ts := event.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	for _, m := range metrics {
		if err := batch.Append(event.TenantID, event.AgentID, m.name, m.value, labels, ts); err != nil {
			return fmt.Errorf("append metric %s: %w", m.name, err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send batch: %w", err)
	}

	return nil
}

// Close closes the ClickHouse connection.
func (w *MetricWriter) Close() error {
	return w.conn.Close()
}
