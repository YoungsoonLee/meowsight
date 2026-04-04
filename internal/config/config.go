package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application.
type Config struct {
	Server     ServerConfig
	Proxy      ProxyConfig
	Postgres   PostgresConfig
	ClickHouse ClickHouseConfig
	Redis      RedisConfig
	NATS       NATSConfig
	S3         S3Config
}

type ServerConfig struct {
	HTTPPort int
	GRPCPort int
}

type ProxyConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxRequestBody  int64
	DefaultProvider string
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

type ClickHouseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type NATSConfig struct {
	URL string
}

type S3Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			HTTPPort: envInt("HTTP_PORT", 8080),
			GRPCPort: envInt("GRPC_PORT", 9090),
		},
		Proxy: ProxyConfig{
			Port:            envInt("PROXY_PORT", 8081),
			ReadTimeout:     envDuration("PROXY_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    envDuration("PROXY_WRITE_TIMEOUT", 120*time.Second),
			MaxRequestBody:  envInt64("PROXY_MAX_REQUEST_BODY", 10<<20), // 10MB
			DefaultProvider: envStr("PROXY_DEFAULT_PROVIDER", "openai"),
		},
		Postgres: PostgresConfig{
			Host:     envStr("POSTGRES_HOST", "localhost"),
			Port:     envInt("POSTGRES_PORT", 5432),
			User:     envStr("POSTGRES_USER", "meowsight"),
			Password: envStr("POSTGRES_PASSWORD", "meowsight"),
			Database: envStr("POSTGRES_DB", "meowsight"),
			SSLMode:  envStr("POSTGRES_SSLMODE", "disable"),
		},
		ClickHouse: ClickHouseConfig{
			Host:     envStr("CLICKHOUSE_HOST", "localhost"),
			Port:     envInt("CLICKHOUSE_PORT", 9000),
			User:     envStr("CLICKHOUSE_USER", "default"),
			Password: envStr("CLICKHOUSE_PASSWORD", ""),
			Database: envStr("CLICKHOUSE_DB", "meowsight"),
		},
		Redis: RedisConfig{
			Addr:     envStr("REDIS_ADDR", "localhost:6379"),
			Password: envStr("REDIS_PASSWORD", ""),
			DB:       envInt("REDIS_DB", 0),
		},
		NATS: NATSConfig{
			URL: envStr("NATS_URL", "nats://localhost:4222"),
		},
		S3: S3Config{
			Endpoint:  envStr("S3_ENDPOINT", "localhost:9000"),
			Bucket:    envStr("S3_BUCKET", "meowsight-audit"),
			AccessKey: envStr("S3_ACCESS_KEY", "minioadmin"),
			SecretKey: envStr("S3_SECRET_KEY", "minioadmin"),
			Region:    envStr("S3_REGION", "us-east-1"),
			UseSSL:    envBool("S3_USE_SSL", false),
		},
	}

	return cfg, nil
}

func envStr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func envInt64(key string, defaultVal int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return defaultVal
}

func envBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultVal
}

func envDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}
