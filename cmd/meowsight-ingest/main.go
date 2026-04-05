package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	chadapter "github.com/YoungsoonLee/meowsight/internal/adapter/clickhouse"
	natsadapter "github.com/YoungsoonLee/meowsight/internal/adapter/nats"
	"github.com/YoungsoonLee/meowsight/internal/config"
	"github.com/YoungsoonLee/meowsight/internal/proxy"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to ClickHouse — metric writer
	metricWriter, err := chadapter.NewMetricWriter(ctx,
		cfg.ClickHouse.Host,
		cfg.ClickHouse.Port,
		cfg.ClickHouse.Database,
		cfg.ClickHouse.User,
		cfg.ClickHouse.Password,
	)
	if err != nil {
		slog.Error("failed to connect to clickhouse for metrics", "error", err)
		os.Exit(1)
	}
	defer metricWriter.Close()

	// Connect to ClickHouse — audit writer
	auditWriter, err := chadapter.NewAuditWriter(ctx,
		cfg.ClickHouse.Host,
		cfg.ClickHouse.Port,
		cfg.ClickHouse.Database,
		cfg.ClickHouse.User,
		cfg.ClickHouse.Password,
	)
	if err != nil {
		slog.Error("failed to connect to clickhouse for audit", "error", err)
		os.Exit(1)
	}
	defer auditWriter.Close()

	// Metric handler: writes metrics to ClickHouse
	metricHandler := func(ctx context.Context, event proxy.RequestEvent) error {
		return metricWriter.WriteMetrics(ctx, event)
	}

	// Audit handler: writes audit log entries to ClickHouse
	auditHandler := func(ctx context.Context, event proxy.RequestEvent) error {
		return auditWriter.WriteAuditLog(ctx, event)
	}

	// Create NATS consumer with both handlers
	consumer, err := natsadapter.NewConsumer(ctx, cfg.NATS.URL, "ingest-writer", metricHandler, auditHandler)
	if err != nil {
		slog.Error("failed to create nats consumer", "error", err)
		os.Exit(1)
	}

	slog.Info("starting meowsight-ingest", "clickhouse", cfg.ClickHouse.Host, "nats", cfg.NATS.URL)

	// Start consumer in background
	go func() {
		if err := consumer.Start(ctx); err != nil && ctx.Err() == nil {
			slog.Error("consumer error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down ingest worker")
	consumer.Stop()
	cancel()
	slog.Info("ingest worker stopped")
}
