package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/YoungsoonLee/meowsight/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	_, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("starting meowsight-ingest")

	// TODO: start gRPC ingest server
	// TODO: start NATS consumers

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("ingest worker stopped")
}
