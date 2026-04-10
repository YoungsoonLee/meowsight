package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	chadapter "github.com/YoungsoonLee/meowsight/internal/adapter/clickhouse"
	pgadapter "github.com/YoungsoonLee/meowsight/internal/adapter/postgres"
	"github.com/YoungsoonLee/meowsight/internal/config"
	"github.com/YoungsoonLee/meowsight/internal/handler/httpapi"
	"github.com/YoungsoonLee/meowsight/internal/migrate"
	"github.com/YoungsoonLee/meowsight/web"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Auto-run migrations on startup so a fresh `docker compose up` is enough
	// to bootstrap the entire stack.
	if err := migrate.RunPostgres(ctx, cfg.Postgres.DSN()); err != nil {
		slog.Error("postgres migrations failed", "error", err)
		os.Exit(1)
	}
	if err := migrate.RunClickHouse(ctx,
		cfg.ClickHouse.Host,
		cfg.ClickHouse.Port,
		cfg.ClickHouse.Database,
		cfg.ClickHouse.User,
		cfg.ClickHouse.Password,
	); err != nil {
		slog.Error("clickhouse migrations failed", "error", err)
		os.Exit(1)
	}

	// Connect to PostgreSQL
	agentRepo, err := pgadapter.NewAgentRepo(ctx, cfg.Postgres.DSN())
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer agentRepo.Close()

	// Connect to ClickHouse
	metricReader, err := chadapter.NewMetricReader(ctx,
		cfg.ClickHouse.Host,
		cfg.ClickHouse.Port,
		cfg.ClickHouse.Database,
		cfg.ClickHouse.User,
		cfg.ClickHouse.Password,
	)
	if err != nil {
		slog.Error("failed to connect to clickhouse", "error", err)
		os.Exit(1)
	}
	defer metricReader.Close()

	// Set up routes
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	dashboard := httpapi.NewDashboardHandler(agentRepo, metricReader)
	dashboard.RegisterRoutes(mux)

	tenantRepo := pgadapter.NewTenantRepo(agentRepo.Pool())
	tenantHandler := httpapi.NewTenantHandler(tenantRepo)
	tenantHandler.RegisterRoutes(mux)

	agentHandler := httpapi.NewAgentHandler(agentRepo)
	agentHandler.RegisterRoutes(mux)

	// Serve embedded web dashboard
	staticFS, _ := fs.Sub(web.StaticFS, "static")
	mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting meowsight-api",
			"port", cfg.Server.HTTPPort,
			"postgres", cfg.Postgres.Host,
			"clickhouse", cfg.ClickHouse.Host,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
