# MeowSight - Claude Code Guidelines

## Project Overview

MeowSight is an AI agent infrastructure management platform built in Go. The MVP is an LLM reverse proxy that captures cost, latency, and audit data with zero agent code changes. Advanced features (SDK, conflict prevention, distributed locking) are planned for the future based on user demand.

## Build & Run

```bash
make build               # Build all binaries
make test                # Run all tests
make lint                # go vet + staticcheck
make run-proxy           # Build + run proxy (port 8081)
make run-api             # Build + run API server (port 8080)
make infra               # docker compose up -d
make infra-down          # docker compose down
make infra-reset         # docker compose down -v (wipes data)
```

## Architecture

Hexagonal architecture with strict dependency rules:

```
handler → app → domain ← adapter
```

- `internal/config/` — App configuration, loaded from env vars with defaults
- `internal/proxy/` — LLM reverse proxy engine (implemented)
- `internal/domain/` — Pure domain types, zero external dependencies
- `internal/app/` — Application services, port interfaces
- `internal/adapter/` — Infrastructure adapters (postgres, clickhouse, redis, nats, s3)
- `internal/handler/` — REST API handlers
- `internal/engine/` — Background processing engines
- `pkg/errors/` — Shared error types
- `configs/` — Runtime config files (pricing.json)
- `cmd/` — Application entry points

## Code Conventions

- **Language**: All code comments, documentation, commit messages in English
- **Go version**: 1.26+
- **Error handling**: Wrap errors with context using `fmt.Errorf("operation: %w", err)`
- **Logging**: Use `log/slog` (stdlib structured logging)
- **Naming**: Follow standard Go conventions (exported = PascalCase, unexported = camelCase)
- **Tests**: Table-driven tests, stdlib `testing` package
- **Context**: Every function touching data takes `context.Context` as first argument
- **Tenant isolation**: All repository methods must include `tenant_id` in queries
- **No hardcoded URLs/values**: Provider URLs via env vars, pricing via `configs/pricing.json`

## Key Binaries

| Binary | Purpose |
|---|---|
| `meowsight-proxy` | LLM reverse proxy (port 8081) — the core product |
| `meowsight-api` | REST API server (port 8080) |
| `meowsight-ingest` | Event ingestion worker |
| `meowsight-worker` | Background job processor |
| `meowctl` | CLI tool |

## Tech Stack

- **Database**: PostgreSQL 16 (config), ClickHouse 24 (metrics/audit), Redis 7 (cache)
- **Message Queue**: NATS JetStream
- **Object Storage**: S3/MinIO (audit archive)
- **CI**: GitHub Actions (go vet + staticcheck)

## LLM Proxy (Implemented)

The proxy is the core product. Key files:

- `internal/proxy/router.go` — Routes `/openai/`, `/anthropic/` to provider handlers
- `internal/proxy/provider/openai.go` — OpenAI reverse proxy (non-streaming + SSE)
- `internal/proxy/provider/anthropic.go` — Anthropic reverse proxy (non-streaming + SSE)
- `internal/proxy/pricing.go` — PricingTable, loads from `configs/pricing.json`
- `internal/proxy/tagger.go` — Extracts tenant/agent ID from `X-Meowsight-*` headers
- `internal/proxy/event.go` — RequestEvent struct (with JSON tags) + EventEmitter interface
- `internal/adapter/nats/emitter.go` — NATS JetStream EventEmitter (production)

### How providers work

- Provider name, prefix, and base URL are injected at construction (not hardcoded)
- Base URLs configurable via env vars (`OPENAI_BASE_URL`, `ANTHROPIC_BASE_URL`)
- Same provider type can be registered multiple times with different names
- OpenAI streaming: `stream_options.include_usage=true` is auto-injected
- Anthropic streaming: parses `message_start` and `message_delta` events for token counts

### Event Pipeline (NATS JetStream)

- `internal/adapter/nats/emitter.go` — Publishes `RequestEvent` to JetStream
- Stream: `EVENTS`, subjects: `events.>`, retention: WorkQueue, max age: 72h
- Subject pattern: `events.{tenant_id}.request`
- Proxy startup: connects to NATS → creates/updates stream → ready
- Fallback: NATS unavailable → auto-fallback to `LogEmitter` (slog)
- Dependencies: `github.com/nats-io/nats.go` (v1.50+)

### ClickHouse Metric Writer

- `internal/adapter/clickhouse/metric_writer.go` — Batch-inserts metrics to ClickHouse `metrics` table
- `internal/adapter/nats/consumer.go` — Durable JetStream consumer, dispatches events to handlers
- `cmd/meowsight-ingest/main.go` — Wires NATS consumer → ClickHouse metric writer
- Metrics per event: `input_tokens`, `output_tokens`, `cost_usd`, `latency_ms`, `error_count` (on error)
- Each metric row has labels: `provider`, `model`, `streaming`
- Consumer: durable `metric-writer`, explicit ack, max 5 retries, 30s ack wait
- Dependencies: `github.com/ClickHouse/clickhouse-go/v2`

### Configuration (env vars)

| Variable | Default |
|---|---|
| `PROXY_PORT` | `8081` |
| `HTTP_PORT` | `8080` |
| `OPENAI_BASE_URL` | `https://api.openai.com` |
| `ANTHROPIC_BASE_URL` | `https://api.anthropic.com` |
| `PRICING_FILE` | `configs/pricing.json` |
| `POSTGRES_*` | `localhost:5432/meowsight` |
| `CLICKHOUSE_*` | `localhost:9000/meowsight` |
| `REDIS_ADDR` | `localhost:6379` |
| `NATS_URL` | `nats://localhost:4222` |

## Database Migrations

- `migrations/postgres/001_init.up.sql` — tenants, agents, budgets, model_pricing, audit_chain
- `migrations/clickhouse/001_init.up.sql` — metrics, audit_log tables

## Important Patterns

- LLM Proxy is the core product — agents change one env var, no code modifications
- External pricing table (`configs/pricing.json`) — no rebuild needed for price changes
- Configurable provider base URLs — supports Azure, local mocks, custom endpoints
- EventEmitter interface decouples proxy from event pipeline (LogEmitter for dev, NATS for production)
- NATS JetStream emitter auto-creates `EVENTS` stream on startup, publishes to `events.{tenant_id}.request`
- Graceful fallback: if NATS is unavailable, proxy falls back to LogEmitter automatically
- Future: SDK/agent integration for distributed locking, conflict prevention (not yet implemented)
