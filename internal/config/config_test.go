package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.HTTPPort != 8080 {
		t.Errorf("expected HTTPPort 8080, got %d", cfg.Server.HTTPPort)
	}
	if cfg.Proxy.Port != 8081 {
		t.Errorf("expected Proxy.Port 8081, got %d", cfg.Proxy.Port)
	}
	if cfg.Postgres.Host != "localhost" {
		t.Errorf("expected Postgres.Host localhost, got %s", cfg.Postgres.Host)
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("expected Redis.Addr localhost:6379, got %s", cfg.Redis.Addr)
	}
	if cfg.NATS.URL != "nats://localhost:4222" {
		t.Errorf("expected NATS.URL nats://localhost:4222, got %s", cfg.NATS.URL)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("HTTP_PORT", "9999")
	os.Setenv("POSTGRES_HOST", "db.example.com")
	defer func() {
		os.Unsetenv("HTTP_PORT")
		os.Unsetenv("POSTGRES_HOST")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.HTTPPort != 9999 {
		t.Errorf("expected HTTPPort 9999, got %d", cfg.Server.HTTPPort)
	}
	if cfg.Postgres.Host != "db.example.com" {
		t.Errorf("expected Postgres.Host db.example.com, got %s", cfg.Postgres.Host)
	}
}

func TestPostgresConfig_DSN(t *testing.T) {
	cfg := PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "meowsight",
		Password: "secret",
		Database: "meowsight",
		SSLMode:  "disable",
	}

	expected := "postgres://meowsight:secret@localhost:5432/meowsight?sslmode=disable"
	if cfg.DSN() != expected {
		t.Errorf("expected DSN %s, got %s", expected, cfg.DSN())
	}
}
