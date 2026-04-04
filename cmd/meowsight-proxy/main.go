package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/YoungsoonLee/meowsight/internal/config"
	"github.com/YoungsoonLee/meowsight/internal/proxy"
	"github.com/YoungsoonLee/meowsight/internal/proxy/provider"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	pricing := proxy.NewPricingTable()
	if err := pricing.LoadFromFile(cfg.Proxy.PricingFile); err != nil {
		slog.Warn("failed to load pricing file, costs will be reported as 0", "path", cfg.Proxy.PricingFile, "error", err)
	}

	emitter := &proxy.LogEmitter{}

	router := proxy.NewRouter(emitter)
	router.RegisterProvider("openai", provider.NewOpenAI("openai", cfg.Proxy.Providers.OpenAIBaseURL, pricing, emitter).Handler())
	router.RegisterProvider("anthropic", provider.NewAnthropic("anthropic", cfg.Proxy.Providers.AnthropicBaseURL, pricing, emitter).Handler())

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Proxy.Port),
		Handler:      router,
		ReadTimeout:  cfg.Proxy.ReadTimeout,
		WriteTimeout: cfg.Proxy.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("starting meowsight-proxy",
			"port", cfg.Proxy.Port,
			"providers", []string{"openai", "anthropic"},
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("proxy server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down proxy")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("proxy shutdown error", "error", err)
	}
	slog.Info("proxy stopped")
}
