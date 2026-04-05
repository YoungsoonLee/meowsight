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

	natsadapter "github.com/YoungsoonLee/meowsight/internal/adapter/nats"
	pgadapter "github.com/YoungsoonLee/meowsight/internal/adapter/postgres"
	"github.com/YoungsoonLee/meowsight/internal/config"
	"github.com/YoungsoonLee/meowsight/internal/proxy"
	"github.com/YoungsoonLee/meowsight/internal/proxy/provider"
	"github.com/jackc/pgx/v5/pgxpool"
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

	var emitter proxy.EventEmitter
	natsEmitter, err := natsadapter.NewEmitter(context.Background(), cfg.NATS.URL)
	if err != nil {
		slog.Warn("failed to connect to NATS JetStream, falling back to log emitter", "error", err)
		emitter = &proxy.LogEmitter{}
	} else {
		slog.Info("using NATS JetStream event emitter")
		emitter = natsEmitter
	}

	// Set up API key resolver (optional — requires PostgreSQL)
	var keyResolver *proxy.KeyResolver
	pool, err := pgxpool.New(context.Background(), cfg.Postgres.DSN())
	if err != nil {
		slog.Warn("failed to connect to PostgreSQL, API key-based auth disabled", "error", err)
	} else {
		keyStore := pgadapter.NewKeyStore(pool)
		keyResolver = proxy.NewKeyResolver(keyStore, 5*time.Minute)
		slog.Info("API key resolver enabled")
	}

	oai := provider.NewOpenAI("openai", cfg.Proxy.Providers.OpenAIBaseURL, pricing, emitter)
	ant := provider.NewAnthropic("anthropic", cfg.Proxy.Providers.AnthropicBaseURL, pricing, emitter)

	if keyResolver != nil {
		oai.SetKeyResolver(keyResolver)
		ant.SetKeyResolver(keyResolver)
	}

	router := proxy.NewRouter(emitter)
	router.RegisterProvider("openai", oai.Handler())
	router.RegisterProvider("anthropic", ant.Handler())

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

	if pool != nil {
		pool.Close()
	}

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("proxy shutdown error", "error", err)
	}
	slog.Info("proxy stopped")
}
