
# 🐱 MeowSight — AI Agent Infrastructure Management

<p align="center">
  <img src="assets/meowsight-logo.png" alt="MeowSight" width="600">
</p>

<p align="center">
  <strong>AI Agent Infrastructure Management Platform</strong>
</p>

<p align="center">
  <em>"Millions of AI agents are running. Who's watching them?"</em><br>
  MeowSight monitors, secures, audits, and controls your AI agents — starting with just one env var change.
</p>

<p align="center">
  <a href="#how-it-works">How It Works</a> &bull;
  <a href="#core-features">Features</a> &bull;
  <a href="#getting-started">Getting Started</a> &bull;
  <a href="#roadmap">Roadmap</a>
</p>

---

> The same logic as selling pickaxes during a gold rush — the more agents there are, the more valuable this infrastructure becomes.

---

## How It Works

MeowSight is an **LLM reverse proxy** that sits between your AI agents and LLM providers. Agents just change one environment variable — no code changes required:

```bash
# Before — agent talks directly to LLM provider
OPENAI_BASE_URL=https://api.openai.com/v1
ANTHROPIC_BASE_URL=https://api.anthropic.com

# After — route through MeowSight proxy
OPENAI_BASE_URL=https://proxy.meowsight.io/openai/v1
ANTHROPIC_BASE_URL=https://proxy.meowsight.io/anthropic/v1
```

The proxy transparently forwards requests while capturing:

- **Token usage and cost** per request (calculated from `configs/pricing.json`)
- **Response latency and error rates**
- **Full request/response audit trail** (configurable)
- **Model and provider breakdown**
- **Per-agent attribution** (via `X-Meowsight-Agent` / `X-Meowsight-Tenant` headers)

```
AI Agents (millions)
    │
    │  LLM API calls (OpenAI, Anthropic, ...)
    ▼
┌────────────────────────────────────────────┐
│         MeowSight LLM Proxy                │
│         (meowsight-proxy)                  │
│                                            │
│  ┌─────────────┐  ┌──────────────────┐     │
│  │ OpenAI      │  │ Anthropic        │     │
│  │ Provider    │  │ Provider         │     │
│  │ (+ Azure,   │  │                  │     │
│  │  compatible)│  │                  │     │
│  └──────┬──────┘  └────────┬─────────┘     │
│         │                  │               │
│         └────────┬─────────┘               │
│                  ▼                         │
│     Extract: tokens, cost, latency         │
│     Emit: RequestEvent                     │
└──────────────────┬─────────────────────────┘
                   │
                   │  NATS JetStream
                   ▼
┌──────────────────────────────────────────┐
│     Event Bus (NATS JetStream)           │
│     subjects: events.{tenant}.{type}     │
└──┬──────────┬──────────┬─────────────────┘
   │          │          │
   ▼          ▼          ▼
 Metric     Audit      Cost
 Writer     Writer     Aggregator
   │          │          │
   ▼          ▼          ▼
ClickH.   ClickH.     PostgreSQL
          + S3
```

**Currently supported providers:** OpenAI, Anthropic (implemented). Planned: Google Gemini, Azure OpenAI, AWS Bedrock, and any OpenAI-compatible API.

---

## Connecting Your Agent

There are **two ways** to connect your AI agent to MeowSight. Choose whichever works best for your setup:

### Option 1: Headers (Recommended)

Best for agents where you control the HTTP client — custom agents, LangChain, CrewAI, AutoGen, etc.

**Step 1:** Point your agent at the MeowSight proxy:
```bash
OPENAI_BASE_URL=https://proxy.meowsight.io/openai/v1
```

**Step 2:** Add identification headers:

```python
# Python (OpenAI SDK)
from openai import OpenAI
client = OpenAI(
    base_url="https://proxy.meowsight.io/openai/v1",
    default_headers={
        "X-Meowsight-Tenant": "my-company",
        "X-Meowsight-Agent": "code-reviewer",
    }
)

# Python (LangChain)
from langchain_openai import ChatOpenAI
llm = ChatOpenAI(
    base_url="https://proxy.meowsight.io/openai/v1",
    default_headers={
        "X-Meowsight-Tenant": "my-company",
        "X-Meowsight-Agent": "summarizer",
    }
)

# Python (Anthropic SDK)
import anthropic
client = anthropic.Anthropic(
    base_url="https://proxy.meowsight.io/anthropic",
    default_headers={
        "X-Meowsight-Tenant": "my-company",
        "X-Meowsight-Agent": "claude-agent",
    }
)

# Node.js (OpenAI SDK)
const client = new OpenAI({
    baseURL: "https://proxy.meowsight.io/openai/v1",
    defaultHeaders: {
        "X-Meowsight-Tenant": "my-company",
        "X-Meowsight-Agent": "support-bot",
    },
});

# cURL
curl https://proxy.meowsight.io/openai/v1/chat/completions \
  -H "Authorization: Bearer sk-your-openai-key" \
  -H "X-Meowsight-Tenant: my-company" \
  -H "X-Meowsight-Agent: test-agent" \
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}]}'
```

### Option 2: API Key (For agents that can't add custom headers)

Best for agents like Cursor, Windsurf, or other tools where you can only change `base_url` and `api_key`.

**Step 1:** Get a MeowSight API key (issued per agent):
```bash
# MeowSight issues a key that maps to your tenant + agent + real LLM key
# Example: ms-cursor-prod-a1b2c3d4e5f6
```

**Step 2:** Use it as your LLM API key:
```bash
# Just change these two settings — no custom headers needed
OPENAI_BASE_URL=https://proxy.meowsight.io/openai/v1
OPENAI_API_KEY=ms-cursor-prod-a1b2c3d4e5f6
```

The proxy automatically:
1. Recognizes the `ms-` key and resolves it to your tenant + agent
2. Swaps it with your real OpenAI/Anthropic API key before forwarding
3. Your agent is identified and tracked without any code changes

### Identification Priority

```
Request arrives at proxy
  │
  ├─ 1. X-Meowsight-* headers present? → Use headers (most flexible)
  │
  ├─ 2. API key recognized (ms-*)? → Resolve from key mapping
  │
  └─ 3. Neither? → Recorded as tenant="default", agent="unknown"
```

| Method | Change needed | Best for |
|---|---|---|
| **Headers** | `base_url` + 2 headers | Custom agents, SDKs, frameworks |
| **API Key** | `base_url` + `api_key` | Cursor, Windsurf, tools with limited config |
| **None** | `base_url` only | Quick testing (no agent attribution) |

---

## Event Pipeline (NATS JetStream)

Every proxied LLM request emits a `RequestEvent` to NATS JetStream for downstream processing:

```
Proxy → NATS JetStream (EVENTS stream) → Consumers (metric writer, audit writer, cost aggregator)
```

### How It Works

1. On proxy startup, the `EVENTS` stream is automatically created (or updated) in JetStream
2. After each LLM request completes, a `RequestEvent` is published to subject `events.{tenant_id}.request`
3. The event contains: tenant/agent IDs, provider, model, token counts, cost, latency, status code, streaming flag, and timestamp
4. Downstream consumers (metric writer, audit writer) subscribe to the stream and process events independently

### Graceful Fallback

If NATS is unavailable (e.g., local development without Docker), the proxy automatically falls back to `LogEmitter` which logs events via `slog`. No configuration change needed — just start the proxy without NATS.

### Stream Configuration

| Setting | Value |
|---|---|
| Stream name | `EVENTS` |
| Subjects | `events.>` |
| Retention | WorkQueue (consumed once per consumer) |
| Max age | 72 hours (replay window) |
| Storage | File-based |

### Event Payload

```json
{
  "tenant_id": "tenant-1",
  "agent_id": "agent-1",
  "provider": "openai",
  "model": "gpt-4o",
  "input_tokens": 100,
  "output_tokens": 50,
  "cost_usd": 0.0075,
  "latency_ms": 230,
  "status_code": 200,
  "streaming": false,
  "timestamp": "2026-04-04T12:00:00Z"
}
```

---

## ClickHouse Metric Writer

The `meowsight-ingest` service consumes events from NATS JetStream and writes metrics to ClickHouse:

```
NATS (EVENTS stream) → meowsight-ingest (metric-writer consumer) → ClickHouse (metrics table)
```

### Metrics per Request

Each proxied LLM request produces the following metrics in ClickHouse:

| Metric | Description |
|---|---|
| `input_tokens` | Number of input/prompt tokens |
| `output_tokens` | Number of output/completion tokens |
| `cost_usd` | Calculated cost in USD |
| `latency_ms` | End-to-end request latency |
| `error_count` | 1 if request had an error (only written on errors) |

Each metric row includes labels: `provider`, `model`, `streaming` — enabling flexible aggregation queries.

### Consumer Configuration

| Setting | Value |
|---|---|
| Consumer name | `metric-writer` |
| Ack policy | Explicit |
| Max deliver | 5 retries |
| Ack wait | 30 seconds |

---

## ClickHouse Audit Writer

The audit writer records every proxied LLM request as a detailed log entry in the `audit_log` table — for compliance, debugging, and forensic analysis.

```
NATS (EVENTS stream) → meowsight-ingest (ingest-writer consumer) → ClickHouse (audit_log table)
```

### Audit Log Fields

| Field | Source | Description |
|---|---|---|
| `id` | Generated UUID | Unique audit entry ID |
| `tenant_id` | `X-Meowsight-Tenant` header | Tenant attribution |
| `agent_id` | `X-Meowsight-Agent` header | Agent attribution |
| `action` | Fixed: `llm_request` | Type of action |
| `resource` | Derived: `/{provider}/v1/chat` | API endpoint |
| `provider` | From proxy | OpenAI, Anthropic, etc. |
| `model` | From LLM response | gpt-4o, claude-sonnet-4-0, etc. |
| `input_tokens` | From LLM response | Prompt token count |
| `output_tokens` | From LLM response | Completion token count |
| `cost_usd` | Calculated | Cost from pricing table |
| `latency_ms` | Measured | Request duration |
| `status_code` | From upstream | HTTP status code |
| `error` | From upstream | Error message (empty on success) |
| `metadata` | Derived | `{"streaming": "true/false"}` |
| `timestamp` | From event | Request timestamp |

### Retention

- **Hot storage**: ClickHouse, 30-day TTL (automatic expiry)
- **Cold storage**: S3 Parquet export (planned, up to 7 years)

---

## Agent Auto-Discovery

Agents are automatically discovered and tracked from proxy traffic — no manual registration required.

```
NATS (EVENTS stream) → meowsight-ingest (agent handler) → PostgreSQL (agents table)
```

### How It Works

1. When a new `tenant_id` + `agent_id` combination appears in proxy traffic, it's automatically registered in PostgreSQL
2. Each subsequent request updates `last_seen_at`, `provider`, `model`, and increments `request_count` (UPSERT)
3. Agent liveness: `WHERE last_seen_at > now() - interval '10 minutes'` → agent is active
4. When a tenant formally registers, existing discovered agents can be linked via `tenant_id` FK

### Agent Lifecycle

```
First request → PostgreSQL agents table (UPSERT)
                  │
                  │ status = 'active', first_seen_at = now()
                  │
               Next requests
                  │
                  │ last_seen_at = now(), request_count++
                  │
               Tenant registers
                  │
                  └→ link tenant_id FK for full management
```

### PostgreSQL `agents` Table

The `agents` table supports both auto-discovery and manual registration:

- **Auto-discovery**: `external_tenant_id` + `external_agent_id` (string, from headers/API key) with UPSERT on conflict
- **Tenant linking**: `tenant_id` FK (nullable) — linked when tenant formally registers
- **Tracking**: `first_seen_at`, `last_seen_at`, `request_count`, `provider`, `model`, `status`
- **Liveness query**: `SELECT * FROM agents WHERE last_seen_at > now() - interval '10 minutes'`

### Running the Ingest Worker

Metric writer, audit writer, and agent tracker all run in the same `meowsight-ingest` process:

```bash
# Requires NATS, ClickHouse, and PostgreSQL to be running
make run-ingest   # or: ./bin/meowsight-ingest
```

---

## REST API (Dashboard Queries)

The `meowsight-api` server (port 8080) provides REST endpoints for dashboard data.

### Endpoints

#### `GET /api/v1/agents` — List discovered agents

```bash
curl "http://localhost:8080/api/v1/agents?tenant_id=my-company"
```

```json
{
  "tenant_id": "my-company",
  "total": 2,
  "agents": [
    {
      "id": "uuid",
      "agent_id": "code-reviewer",
      "status": "active",
      "provider": "openai",
      "model": "gpt-4o",
      "first_seen_at": "2026-04-01T10:00:00Z",
      "last_seen_at": "2026-04-05T12:30:00Z",
      "request_count": 1523,
      "active": true
    }
  ]
}
```

`active` field: `true` if `last_seen_at` is within the last 10 minutes.

#### `GET /api/v1/metrics/summary` — Aggregated metrics

```bash
curl "http://localhost:8080/api/v1/metrics/summary?tenant_id=my-company&from=2026-04-01T00:00:00Z&to=2026-04-05T23:59:59Z"
```

```json
{
  "tenant_id": "my-company",
  "total_cost_usd": 12.45,
  "total_input_tokens": 450000,
  "total_output_tokens": 120000,
  "breakdown": [
    {
      "agent_id": "code-reviewer",
      "provider": "openai",
      "model": "gpt-4o",
      "total_cost_usd": 8.50,
      "avg_latency_ms": 230,
      "request_count": 1200
    }
  ]
}
```

Default time range: last 24 hours. Use `from` and `to` (RFC3339) to customize.

#### `GET /api/v1/audit` — Audit log entries

```bash
curl "http://localhost:8080/api/v1/audit?tenant_id=my-company&limit=10&offset=0"
```

```json
{
  "tenant_id": "my-company",
  "limit": 10,
  "offset": 0,
  "logs": [
    {
      "id": "uuid",
      "agent_id": "code-reviewer",
      "provider": "openai",
      "model": "gpt-4o",
      "input_tokens": 150,
      "output_tokens": 80,
      "cost_usd": 0.0012,
      "latency_ms": 245,
      "status_code": 200,
      "timestamp": "2026-04-05T12:30:00Z"
    }
  ]
}
```

#### `GET /healthz` — Health check

```bash
curl http://localhost:8080/healthz
# {"status":"ok"}
```

### Running the API Server

```bash
# Requires PostgreSQL and ClickHouse to be running
make run-api   # or: ./bin/meowsight-api (port 8080)
```

---

## Core Features

| Domain | Description | How |
|---|---|---|
| **Agent Monitoring** | Latency, error rates, request volume, agent liveness | Extracted from proxy traffic |
| **Security** | Model/provider allowlists, rate limiting, content filtering | Enforced at proxy layer |
| **Audit Trail** | Full LLM request/response logging, tamper-proof records | Stored in ClickHouse + S3 |
| **Cost Management** | Token counting, cost calculation, budget enforcement | Pricing table + real-time aggregation |

---

## System Architecture

### Data Flow

1. Agents route LLM API calls through the MeowSight proxy — cost, latency, and audit data are captured transparently
2. The proxy emits `RequestEvent`s to NATS JetStream
3. Domain-specific consumers process events and write to appropriate storage
4. The proxy can enforce budgets by rejecting requests when spend exceeds limits

### Storage Strategy

| Store | Technology | Purpose | Retention |
|---|---|---|---|
| Config DB | PostgreSQL | Tenants, agents, policies, budgets | Permanent |
| Metrics | ClickHouse | Time-series metrics | 90 days hot, 1 year cold |
| Audit Hot | ClickHouse | Recent audit logs | 30 days |
| Audit Cold | S3 (Parquet) | Long-term audit logs | Up to 7 years |
| Cache | Redis Cluster | Real-time status, rate limits | Ephemeral |
| Event Bus | NATS JetStream | Inter-service events | 72h replay window |

### Multi-Tenant Architecture

- **PostgreSQL:** Row-Level Security (RLS) + `tenant_id` column
- **ClickHouse:** Partitioned by `(tenant_id, toYYYYMM(timestamp))`
- **NATS:** Subject hierarchy `events.{tenant_id}.{event_type}`

---

## Tech Stack

| Layer | Technology | Rationale |
|---|---|---|
| Language | **Go** | High performance, concurrency, single binary deployment |
| HTTP | `net/http` + `chi/v5` | Lightweight, composable middleware |
| PostgreSQL | `pgx/v5` | Config, tenants, policies, budgets |
| ClickHouse | `clickhouse-go/v2` | Metrics + audit logs (high-volume time-series) |
| Redis | `go-redis/v9` | Cache, rate limiting |
| Message Queue | **NATS JetStream** | Low latency, simple operations, at-least-once delivery |
| Object Storage | S3 / MinIO | Long-term audit log archive (Parquet) |
| Deployment | Kubernetes + Helm | Production orchestration |

---

## Project Structure

```
meowsight/
├── cmd/
│   ├── meowsight-api/                # REST API server
│   ├── meowsight-proxy/              # LLM reverse proxy
│   ├── meowsight-ingest/             # Event ingestion worker
│   ├── meowsight-worker/             # Background job processor
│   └── meowctl/                      # CLI tool
│
├── configs/
│   └── pricing.json                  # Model pricing table (edit without rebuild)
│
├── internal/
│   ├── config/                       # App configuration (env vars)
│   ├── proxy/                        # LLM proxy engine Implemented
│   │   ├── router.go                 # Route requests to correct LLM provider
│   │   ├── event.go                  # RequestEvent struct + EventEmitter interface
│   │   ├── event_logger.go           # Dev-mode event logger (slog)
│   │   ├── pricing.go                # PricingTable — loads from configs/pricing.json
│   │   ├── tagger.go                 # Agent attribution from X-Meowsight-* headers
│   │   └── provider/
│   │       ├── openai.go             # OpenAI / OpenAI-compatible (streaming + non-streaming)
│   │       └── anthropic.go          # Anthropic Claude (streaming + non-streaming)
│   ├── domain/                       # Pure domain types
│   ├── app/                          # Application services (use cases + ports)
│   ├── adapter/                      # Infrastructure adapters (postgres, clickhouse, redis, nats, s3)
│   ├── handler/                      # REST API handlers
│   └── engine/                       # Background engines (alerting, cost, archiver)
│
├── pkg/
│   └── errors/                       # Shared error types
│
├── migrations/
│   ├── postgres/001_init.up.sql      # Tenants, agents, budgets, model_pricing
│   └── clickhouse/001_init.up.sql    # Metrics + audit_log tables
│
├── deploy/docker/                    # Multi-stage Dockerfiles
├── .github/workflows/ci.yml         # CI: test + lint
├── docker-compose.yml               # Local dev: PG, ClickHouse, Redis, NATS, MinIO
├── Makefile
└── go.mod
```

---

## Getting Started

```bash
# Start local infrastructure
docker-compose up -d  # PostgreSQL, ClickHouse, Redis, NATS, MinIO

# Build all binaries
make build

# Run tests
make test

# Run the LLM proxy (default port 8081)
./bin/meowsight-proxy

# Run the API server (default port 8080)
./bin/meowsight-api
```

### Configuration

All configuration is via environment variables with sensible defaults:

| Variable | Default | Description |
|---|---|---|
| `PROXY_PORT` | `8081` | LLM proxy listen port |
| `HTTP_PORT` | `8080` | API server listen port |
| `OPENAI_BASE_URL` | `https://api.openai.com` | OpenAI upstream URL |
| `ANTHROPIC_BASE_URL` | `https://api.anthropic.com` | Anthropic upstream URL |
| `PRICING_FILE` | `configs/pricing.json` | Path to model pricing table |
| `POSTGRES_HOST` | `localhost` | PostgreSQL host |
| `CLICKHOUSE_HOST` | `localhost` | ClickHouse host |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |

### Model Pricing

Model pricing is managed in `configs/pricing.json` — no code changes or rebuilds needed:

```json
{
  "models": {
    "gpt-4o": {"provider": "openai", "input_per_1k": 0.0025, "output_per_1k": 0.01},
    "claude-sonnet-4-0": {"provider": "anthropic", "input_per_1k": 0.003, "output_per_1k": 0.015}
  }
}
```

---

## Roadmap

### v0.1 — LLM Proxy Core

- [x] Project scaffolding, Go module, `Makefile`, `.gitignore` ✅
- [x] Docker Compose (PostgreSQL, ClickHouse, Redis, NATS, MinIO) ✅
- [x] CI pipeline (`go vet` + `staticcheck`) ✅
- [x] Multi-stage Dockerfiles (api, proxy, ingest, worker) ✅
- [x] DB migrations (PostgreSQL + ClickHouse) ✅
- [x] OpenAI reverse proxy (non-streaming + SSE streaming) ✅
- [x] Anthropic reverse proxy (non-streaming + SSE streaming) ✅
- [x] Token/cost extraction from LLM responses ✅
- [x] External pricing table (`configs/pricing.json`) ✅
- [x] Configurable provider base URLs via env vars ✅
- [x] Agent attribution via `X-Meowsight-*` headers ✅
- [x] Auto-inject `stream_options.include_usage` for OpenAI streaming ✅

### v0.2 — Event Pipeline & Dashboard

- [x] Event emission to NATS JetStream ✅
- [x] ClickHouse metric writer (latency, tokens, errors) ✅
- [x] ClickHouse audit writer (request/response logs) ✅
- [x] Agent auto-discovery from proxy traffic ✅
- [x] REST API for dashboard queries ✅
- [ ] Web dashboard (cost trends, agent status, audit logs)
- [x] API key authentication for tenants ✅
- [ ] Tenant registration and management
- [ ] All-in-one `docker compose up` for full local deployment (proxy, api, ingest, worker + infra)

### v0.3 — Budget & Security

- [ ] Budget enforcement (reject requests on overage)
- [ ] Model/provider allowlists per tenant
- [ ] Per-agent cost dashboard
- [ ] Cost anomaly alerts (webhook / email)
- [ ] Error rate spike detection
- [ ] Agent-down detection
- [ ] `meowctl top` — real-time CLI dashboard (agent traffic, cost, latency, errors)
- [ ] Content filtering
- [ ] PII masking in audit logs

### v0.4 — More Providers & Hardening

- [ ] Google Gemini provider
- [ ] Azure OpenAI provider
- [ ] AWS Bedrock provider
- [ ] Any OpenAI-compatible API support
- [ ] RBAC for dashboard
- [ ] Threat detection v1 (runaway agent / cost spike)
- [ ] Audit archiver (S3 Parquet export)
- [ ] Multi-tenant hardening (RLS, per-tenant rate limiting)

### v1.0 — SaaS Launch

- [ ] Stripe billing integration (subscriptions, usage-based overage)
- [ ] Plan enforcement and usage metering
- [ ] Kubernetes Helm charts + HPA
- [ ] Production deployment
- [ ] Documentation site
- [ ] Onboarding flow
- [ ] `meowctl doctor` — self-diagnosis tool

> **Future:** SDK and agent-side integration (Go, Python, TypeScript) for distributed locking, intent-based conflict prevention, and server-push directives — added based on user demand after proxy MVP is validated.

---

## Key Design Decisions

| Decision | Rationale |
|---|---|
| LLM Proxy as MVP | Zero-code integration (env var change only) maximizes adoption; covers monitoring, security, audit, and cost without agent modifications |
| External pricing table (JSON) | Model prices change frequently; JSON file avoids code changes and rebuilds, upgradeable to DB later |
| Configurable provider base URLs | Supports Azure OpenAI, local mocks, and custom endpoints via env vars without code changes |
| Separate proxy and API binaries | Independent scaling; dashboard queries don't starve proxy traffic |
| ClickHouse for metrics and audit | Reduced operational complexity, same query patterns, splittable later |
| NATS over Kafka | Lower latency, simpler operations, swappable via adapter pattern |

---

## Why "MeowSight"?

Cats see in the dark. MeowSight gives you visibility into the invisible — what your AI agents are doing, how much they cost, and whether they're behaving. Like a cat watching from the shadows, it observes everything without getting in the way.

---

## Contributing

AI/vibe-coded PRs welcome!

---

## License

MIT
