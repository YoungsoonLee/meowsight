# MeowSight - Claude Code Guidelines

## Project Overview

MeowSight is an AI agent infrastructure management platform built in Go. It provides monitoring, security, audit trails, cost management, and conflict prevention for AI agents at scale.

## Build & Run

```bash
# Initialize module
go mod init github.com/YoungsoonLee/meowsight

# Build all binaries
make build

# Run tests
make test

# Run specific test
go test ./internal/...

# Local infrastructure
docker-compose up -d   # PostgreSQL, ClickHouse, Redis, NATS

# Run services
./bin/meowsight-api
./bin/meowsight-proxy
./bin/meowsight-ingest
./bin/meowsight-worker
```

## Architecture

Hexagonal architecture with strict dependency rules:

```
handler → app → domain ← adapter
```

- `internal/domain/` — Pure domain types, zero external dependencies
- `internal/app/` — Application services, port interfaces (use cases)
- `internal/adapter/` — Infrastructure adapters implementing ports
- `internal/proxy/` — LLM reverse proxy engine (provider adapters, token extraction, streaming)
- `internal/handler/` — Inbound adapters (HTTP, gRPC, ingestion)
- `internal/engine/` — Background processing engines
- `pkg/sdk/` — Public Go SDK for agent integration
- `cmd/` — Application entry points

## Code Conventions

- **Language**: All code comments, documentation, commit messages in English
- **Go version**: 1.22+
- **Error handling**: Wrap errors with context using `fmt.Errorf("operation: %w", err)`
- **Logging**: Use `log/slog` (stdlib structured logging)
- **Naming**: Follow standard Go conventions (exported = PascalCase, unexported = camelCase)
- **Tests**: Table-driven tests, use `testify` for assertions
- **Context**: Every function touching data takes `context.Context` as first argument
- **Tenant isolation**: All repository methods must include `tenant_id` in queries — never skip this

## Key Binaries

| Binary | Purpose |
|---|---|
| `meowsight-api` | REST + gRPC API server |
| `meowsight-proxy` | LLM reverse proxy — zero-code agent integration (MVP killer feature) |
| `meowsight-ingest` | High-throughput event ingestion worker |
| `meowsight-worker` | Background job processor (alerting, archiving, cost aggregation) |
| `meowctl` | CLI tool for operators |

## Tech Stack

- **Database**: PostgreSQL (config), ClickHouse (metrics/audit), Redis (cache/locks)
- **Message Queue**: NATS JetStream
- **Object Storage**: S3/MinIO (audit archive)
- **Protobuf**: Managed with `buf.build`
- **HTTP Router**: `chi/v5`
- **gRPC**: `google.golang.org/grpc`

## Important Patterns

- **2-tier agent integration**: Zero-code (LLM Proxy) → Full-code (SDK/gRPC); OTel as optional ingestion format
- LLM Proxy is the MVP killer feature — agents change one env var, covers 4/5 domains, no code modifications
- Phase 1 is Proxy-only; SDK comes in Phase 3 for conflict prevention
- Micro-batching for event ingestion (flush every 100ms or 1000 events)
- Redis Redlock for distributed locking
- SHA-256 chain hashing for audit trail tamper evidence
- gRPC bidirectional streaming for agent communication
- Feature gating via tenant plan loaded from Redis cache
