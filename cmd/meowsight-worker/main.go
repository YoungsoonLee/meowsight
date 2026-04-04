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

	slog.Info("starting meowsight-worker")

	// TODO: start background engines
	// - alerting evaluator
	// - cost aggregator
	// - audit archiver
	// - threat detector

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("worker stopped")
}
