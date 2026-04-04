
# 🐱 MeowSight — AI Agent Infrastructure Management Platform

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
  <a href="#tier-1-llm-proxy-zero-code--mvp-killer-feature">Quick Start</a> &bull;
  <a href="#core-features">Features</a> &bull;
  <a href="#mvp-roadmap">Roadmap</a> &bull;
  <a href="#getting-started">Getting Started</a>
</p>

---

> The same logic as selling pickaxes during a gold rush — the more agents there are, the more valuable this infrastructure becomes.

---

## Core Features

| Domain | Description |
|---|---|
| **Agent Monitoring** | Agent health, performance metrics, alerting |
| **Security** | Authentication/authorization, threat detection, policy-based access control |
| **Audit Trail** | Full agent action logging, compliance, tamper-proof records |
| **Cost Management** | Token usage tracking, budget limits, cost allocation |
| **Conflict Prevention** | Distributed locks, intent-based conflict detection, agent coordination |

---

## Agent Integration — 2-Tier Strategy

Existing agents should be monitorable **without code changes**. MeowSight provides two integration tiers:

```
┌───────────────────────────────────────────────────────────┐
│                    Integration Tiers                      │
├─────────────────────────────┬─────────────────────────────┤
│  Tier 1: LLM Proxy          │  Tier 2: SDK                │
│  (Zero-Code)                │  (Full-Code)                │
├─────────────────────────────┼─────────────────────────────┤
│  Change: env var only       │  Change: import SDK         │
│                             │                             │
│  ✅ Token usage & cost      │  ✅ Everything in Tier 1    │
│  ✅ Latency & error rates   │  ✅ Distributed locks       │
│  ✅ Audit trail (LLM calls) │  ✅ Intent-based conflict   │
│  ✅ Budget enforcement      │  ✅ Server push directives  │
│  ✅ Model/provider control  │  ✅ Non-LLM action tracking │
│  ✅ Per-agent attribution   │  ✅ Custom metrics          │
│                             │                             │
│  Effort: ★☆☆☆☆              │  Effort: ★★★☆☆              │
│  Covers: 4/5 domains        │  Covers: 5/5 domains        │
└─────────────────────────────┴─────────────────────────────┘

  * OTel is supported as an ingestion format, not a separate tier.
    Teams already using OpenTelemetry can export to MeowSight's
    OTLP endpoint without adopting the SDK.
```

### Tier 1: LLM Proxy (Zero-Code) — MVP Killer Feature

A reverse proxy that sits between agents and LLM providers. Agents just change one environment variable:

```bash
# Before — agent talks directly to LLM provider
OPENAI_BASE_URL=https://api.openai.com/v1
ANTHROPIC_BASE_URL=https://api.anthropic.com

# After — route through MeowSight proxy
OPENAI_BASE_URL=https://proxy.meowsight.io/openai/v1
ANTHROPIC_BASE_URL=https://proxy.meowsight.io/anthropic/v1
```

The proxy transparently forwards requests while capturing:
- Token usage and cost per request
- Response latency and error rates
- Full request/response audit trail (configurable)
- Model and provider breakdown
- Per-agent attribution (via API key or `X-MeowSight-Agent` header)

**Supported providers:** OpenAI, Anthropic, Google Gemini, Azure OpenAI, AWS Bedrock, Cohere, Mistral, and any OpenAI-compatible API.

**What Tier 1 covers:**

| Domain | Proxy Coverage |
|---|---|
| Monitoring | ✅ Latency, error rates, request volume, agent liveness via call patterns |
| Security | ✅ Model/provider allowlists, rate limiting, content filtering |
| Audit Trail | ✅ Full LLM request/response logging |
| Cost Management | ✅ Token counting, cost calculation, budget enforcement (block requests on overage) |
| Conflict Prevention | ❌ Requires SDK (Tier 2) |

### Tier 2: SDK / gRPC (Full-Code)

For advanced features that require agent-side integration: distributed locking, intent-based conflict detection, non-LLM action tracking, and server-push directives. See [SDK Usage Example](#sdk-usage-example) below.

The natural adoption path: start with Tier 1 (zero friction), then upgrade to Tier 2 when conflict prevention or deeper observability is needed.

---

## System Architecture

```
AI Agents (millions)
    │
    ├── Tier 1: LLM API calls routed through proxy (most agents)
    └── Tier 2: gRPC streaming via SDK (advanced agents)
    │
    ▼
┌────────────────────────────────────────────────────┐
│              MeowSight Ingestion                   │
│                                                    │
│  ┌──────────────────┐      ┌───────────────────┐   │
│  │ LLM Proxy        │      │ gRPC/HTTP Ingest  │   │
│  │ (meowsight-proxy)│      │ (meowsight-ingest)│   │
│  │                  │      │                   │   │
│  │ Intercepts LLM   │      │ SDK events +      │   │
│  │ traffic, extracts│      │ OTel (optional)   │   │
│  │ cost/latency/    │      │                   │   │
│  │ audit data       │      │                   │   │
│  └────────┬─────────┘      └─────────┬─────────┘   │
│           │                          │             │
│           └──────────┬───────────────┘             │
│                      ▼                             │
│            Unified Event Pipeline                  │
└──────────────────────┬─────────────────────────────┘
                       │
                       │  NATS JetStream
                       ▼
┌──────────────────────────────────────────────────┐
│              Event Bus (NATS JetStream)          │
│              subjects: events.{tenant}.{type}    │
└──┬──────────┬──────────┬──────────┬──────────────┘
   │          │          │          │
   ▼          ▼          ▼          ▼
 Metric     Audit      Cost     Conflict
 Writer     Writer     Agg.     Detector
   │          │          │          │
   ▼          ▼          ▼          ▼
ClickH.   ClickH.     PG      Redis+PG
          + S3
```

### Data Flow

1. **Tier 1 (most agents):** LLM API calls are routed through the MeowSight proxy — cost, latency, and audit data are captured transparently with zero agent code changes
2. **Tier 2 (advanced agents):** SDK-integrated agents connect via gRPC bidirectional streaming for full feature access including conflict prevention
3. **OTel (optional):** Teams already using OpenTelemetry can export to MeowSight's OTLP endpoint via the ingest service
4. Both tiers feed into a unified event pipeline, then publish to NATS
5. Domain-specific consumers process events and write to appropriate storage
6. The proxy can enforce budgets by rejecting requests when spend exceeds limits
7. For SDK-connected agents, the server can additionally push policy updates, PAUSE directives, and kill signals via the return stream

---

## Tech Stack

| Layer | Technology | Rationale |
|---|---|---|
| Language | **Go** | High performance, concurrency, single binary deployment |
| HTTP | `chi/v5` | Lightweight, composable middleware |
| gRPC | `google.golang.org/grpc` | High-performance protocol for agent communication |
| PostgreSQL | `pgx/v5` | Config, tenants, policies, budgets |
| ClickHouse | `clickhouse-go/v2` | Metrics + audit logs (high-volume time-series) |
| Redis | `go-redis/v9` | Cache, distributed locks, rate limiting |
| Message Queue | **NATS JetStream** | Low latency, simple operations, at-least-once delivery |
| Object Storage | S3 / MinIO | Long-term audit log archive (Parquet) |
| Observability | OpenTelemetry + Prometheus | Self-monitoring |
| Deployment | Kubernetes + Helm | Production orchestration |

---

## Project Structure

Based on hexagonal architecture.

```
meowsight/
├── cmd/                              # Application entry points
│   ├── meowsight-api/                # REST + gRPC API server
│   │   └── main.go
│   ├── meowsight-proxy/              # LLM reverse proxy (zero-code integration)
│   │   └── main.go
│   ├── meowsight-ingest/             # High-throughput event ingestion worker
│   │   └── main.go
│   ├── meowsight-worker/             # Background job processor
│   │   └── main.go
│   └── meowctl/                      # CLI tool
│       └── main.go
│
├── internal/                         # Private application code
│   ├── domain/                       # Pure domain types (no external dependencies)
│   │   ├── agent/                    # Agent, AgentGroup, AgentStatus
│   │   ├── monitoring/               # Metric, AlertRule, Alert
│   │   ├── security/                 # Policy, Permission, ThreatEvent
│   │   ├── audit/                    # AuditRecord, AuditQuery
│   │   ├── cost/                     # TokenUsage, Budget, CostAllocation
│   │   ├── conflict/                 # ResourceLock, ConflictEvent
│   │   └── tenant/                   # Tenant, Workspace, Plan
│   │
│   ├── app/                          # Application services (use cases + ports)
│   │   ├── monitoring/               # service.go + ports.go
│   │   ├── security/
│   │   ├── audit/
│   │   ├── cost/
│   │   ├── conflict/
│   │   └── tenant/
│   │
│   ├── adapter/                      # Infrastructure adapters (port implementations)
│   │   ├── postgres/                 # PostgreSQL repositories + migrations
│   │   ├── clickhouse/               # ClickHouse repositories
│   │   ├── redis/                    # Locks, cache, rate limiting
│   │   ├── nats/                     # Publisher, Subscriber
│   │   ├── s3/                       # Audit log archive
│   │   └── notification/             # Slack, PagerDuty, Webhook
│   │
│   ├── proxy/                        # LLM proxy engine
│   │   ├── router.go                 # Route requests to correct LLM provider
│   │   ├── provider/                 # Per-provider adapters
│   │   │   ├── openai.go             # OpenAI / OpenAI-compatible
│   │   │   ├── anthropic.go          # Anthropic Claude
│   │   │   ├── google.go             # Google Gemini
│   │   │   └── bedrock.go            # AWS Bedrock
│   │   ├── interceptor.go            # Extract tokens, cost, latency from responses
│   │   ├── streamer.go               # SSE/streaming response handling
│   │   └── tagger.go                 # Agent attribution from API key / headers
│   │
│   ├── handler/                      # Inbound adapters
│   │   ├── httpapi/                  # REST API handlers
│   │   ├── grpcapi/                  # gRPC services (agent SDK facing)
│   │   └── ingest/                   # High-throughput event ingestion handler
│   │
│   ├── engine/                       # Background processing engines
│   │   ├── alerting/                 # Alert rule evaluation
│   │   ├── costengine/               # Cost aggregation and budget enforcement
│   │   ├── conflictdetector/         # Conflict detection and resolution
│   │   ├── threatdetector/           # Anomalous behavior detection
│   │   └── archiver/                 # Audit log S3 archiver
│   │
│   └── config/                       # App configuration
│
├── pkg/                              # Public packages
│   ├── sdk/                          # MeowSight Go SDK
│   ├── proto/                        # Generated protobuf code
│   ├── middleware/                    # Shared middleware
│   └── errors/                       # Shared error types
│
├── api/                              # API specifications
│   ├── openapi/                      # OpenAPI 3.1 spec
│   └── proto/                        # .proto source files
│
├── deploy/                           # Deployment manifests
│   ├── kubernetes/
│   ├── helm/
│   └── docker/
│
├── migrations/                       # DB migrations
│   ├── postgres/
│   └── clickhouse/
│
├── scripts/
├── docs/
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### Dependency Rule

```
handler → app → domain ← adapter
```

- `domain`: No external dependencies (pure business logic)
- `app`: Imports only domain, defines port interfaces
- `adapter`: Imports domain + external drivers, implements port interfaces
- `handler`: Calls app services

---

## Domain Details

### 1. Agent Monitoring

**Core Entities:**
- `Agent` — id, name, group, tenant, status (online/degraded/offline), last_heartbeat
- `Metric` — agent_id, timestamp, metric_name, value, labels
- `AlertRule` — condition expression (PromQL-like), threshold, duration, notification channels
- `Alert` — status (firing/resolved), fired_at, resolved_at

**How It Works:**
- Heartbeat → Redis (real-time status) + ClickHouse (history)
- Agent offline detection: Redis TTL expiry → NATS event → alert
- Alert engine: queries ClickHouse every 15s, fires notifications when conditions are met

**Alert Condition DSL:**
```
avg(agent_latency{group="order-processors"}) > 500 for 5m
count(agent_status == "offline") / count(agent_status) > 0.1 for 2m
```

### 2. Security

**Authentication:**
- Users: JWT (self-issued or OIDC/SAML federation)
- Agents: API keys (`ms_live_xxx`), scoped permissions, stored as SHA-256 hash

**Authorization:**
- RBAC: tenant → workspace → agent_group (3 levels)
- Per-agent policies: "Agent X may only call OpenAI API"
- Policies pushed to agents via gRPC HeartbeatResponse

**Threat Detection:**
- Event volume spikes (runaway agent)
- Out-of-scope action patterns
- Cost anomalies (token usage beyond 3σ)

### 3. Audit Trail

**Storage Tiering:**
1. Events → NATS → ClickHouse (30-day hot storage)
2. Daily archiver: export to S3 as Parquet (up to 7-year retention)
3. Queries: hot storage first, fan-out to S3 for extended time ranges

**Tamper Prevention:**
- No `ALTER DELETE` permission for ClickHouse app user
- Per-batch SHA-256 chain hash (previous batch hash + current content hash) → stored in PostgreSQL

### 4. Cost Management

**Core Entities:**
- `TokenUsage` — agent_id, model, provider, input/output tokens, estimated_cost_usd
- `Budget` — scope (tenant/workspace/group/agent), period (daily/weekly/monthly), limit_usd

**Budget Enforcement:**
1. Agent reports usage → cost aggregator updates real-time totals
2. Limit exceeded → NATS `budget.exceeded` event published
3. gRPC server sends `PAUSE` directive via bidirectional stream

**Cost Calculation:**
- `model_pricing` table (model, provider, input/output unit price, effective_date)
- Server-side calculation or pre-calculated cost from SDK

### 5. Conflict Prevention

**Distributed Locking:**
- Redis + Redlock algorithm
- Hierarchical resource keys: `orders/customer-456`, `database/table-users/row-123`
- Mandatory TTL (max 5 min), heartbeat-based extension, auto-expiry on crash

**Intent-Based Conflict Detection:**
- Agents declare intended actions before execution: "I intend to modify customer 456's order"
- Conflict matching engine cross-references pending intents
- Resolution strategies: mutex (one wins), queue (FIFO), merge (both proceed), escalate (notify human)

---

## API Design

### REST API (Dashboard / Management)

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/agents` | GET/POST | List/register agents |
| `/api/v1/agents/{id}/status` | GET | Current agent status |
| `/api/v1/agents/{id}/metrics` | GET | Query metrics (time range) |
| `/api/v1/alerts` | GET/POST/PUT | Manage alert rules |
| `/api/v1/audit/search` | POST | Search audit logs |
| `/api/v1/cost/usage` | GET | Usage summary |
| `/api/v1/cost/budgets` | GET/POST/PUT | Budget management |
| `/api/v1/policies` | CRUD | Security policy management |
| `/api/v1/conflicts` | GET | Active conflicts |
| `/api/v1/locks` | GET/POST/DELETE | Resource lock management |

### gRPC API (Agent-Facing)

```protobuf
service AgentIngestService {
  rpc EventStream(stream AgentEvent) returns (stream ServerDirective);
  rpc IngestBatch(EventBatch) returns (IngestResponse);
}

service AgentControlService {
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  rpc AcquireLock(LockRequest) returns (LockResponse);
  rpc ReleaseLock(ReleaseRequest) returns (ReleaseResponse);
  rpc CheckConflict(ConflictCheckRequest) returns (ConflictCheckResponse);
}
```

---

## SDK Usage Example

```go
import "github.com/YoungsoonLee/meowsight/pkg/sdk"

ms, _ := meowsight.New(
    meowsight.WithAPIKey("ms_live_xxx"),
    meowsight.WithAgentID("order-processor-1"),
    meowsight.WithAgentGroup("order-processors"),
)
defer ms.Close()

// Automatic heartbeats start (every 10s)

// Record an action
ms.RecordAction("process_order", map[string]string{"order_id": "123"})

// Report cost
ms.ReportUsage(meowsight.Usage{
    Model: "claude-4", InputTokens: 500, OutputTokens: 200,
})

// Acquire a distributed lock
lock, _ := ms.AcquireLock("orders/customer-456", 30*time.Second)
defer lock.Release()
```

---

## Multi-Tenant Architecture

```
Request → Middleware(TenantContext)
            │
            ├── Extract tenant_id from API key / JWT
            ├── Load tenant plan from Redis (cached)
            ├── Inject into context.Context
            └── Apply per-plan rate limits

All repository methods extract tenant_id from context and include it in every query.
```

- **PostgreSQL:** Row-Level Security (RLS) + `tenant_id` column
- **ClickHouse:** Partitioned by `(tenant_id, toYYYYMM(timestamp))`
- **NATS:** Subject hierarchy `events.{tenant_id}.{event_type}`

---

## Storage Strategy

| Store | Technology | Purpose | Retention |
|---|---|---|---|
| Config DB | PostgreSQL | Tenants, agents, policies, budgets | Permanent |
| Metrics | ClickHouse | Time-series metrics | 90 days hot, 1 year cold |
| Audit Hot | ClickHouse | Recent audit logs | 30 days |
| Audit Cold | S3 (Parquet) | Long-term audit logs | Up to 7 years |
| Cache/Lock | Redis Cluster | Real-time status, locks, rate limits | Ephemeral |
| Event Bus | NATS JetStream | Inter-service events | 72h replay window |

---

## MVP Roadmap

### Phase 1: LLM Proxy MVP (Weeks 1-8)

The proxy alone delivers 4/5 core domains. Ship this first, get users, then expand.

| Week | Deliverable |
|---|---|
| 1-2 | Project scaffolding, Go module, Docker Compose (PG, ClickHouse, Redis, NATS), CI setup |
| 3-4 | **LLM Proxy core**: reverse proxy for OpenAI + Anthropic, request forwarding, SSE streaming support, token/cost extraction from responses |
| 5-6 | **Proxy → pipeline**: event emission to NATS, ClickHouse metric/audit writers, agent auto-discovery (identify agents by API key or `X-MeowSight-Agent` header) |
| 7-8 | **Proxy features**: budget enforcement (reject on overage), model/provider allowlists, per-agent cost dashboard, API key auth, tenant registration, REST API for dashboard queries |

**Phase 1 outcome:** A working product where users change one env var and immediately get cost tracking, latency monitoring, audit logs, and budget enforcement.

### Phase 2: Proxy Hardening + Additional Providers (Weeks 9-14)

| Week | Deliverable |
|---|---|
| 9-10 | Additional providers: Google Gemini, Azure OpenAI, AWS Bedrock, OpenAI-compatible APIs |
| 11-12 | Alert engine: cost anomaly alerts, error rate spikes, agent-down detection, webhook/email notifications |
| 13-14 | Security: content filtering, PII masking in audit logs, RBAC for dashboard, threat detection v1 (runaway agent / cost spike) |

### Phase 3: SDK + Conflict Prevention (Weeks 15-22)

| Week | Deliverable |
|---|---|
| 15-16 | Go SDK: gRPC client, heartbeat, event reporting, action wrapper |
| 17-18 | Distributed locking: Redis Redlock, lock API, SDK lock client |
| 19-20 | Intent-based conflict detection, resolution strategies (mutex, queue) |
| 21-22 | Audit archiver (S3 Parquet), advanced threat detection, Python/TypeScript SDK v1 |

### Phase 4: SaaS Launch (Weeks 23-30)

| Week | Deliverable |
|---|---|
| 23-24 | Multi-tenant hardening: RLS, per-tenant rate limiting, plan enforcement, usage metering |
| 25-26 | Stripe billing integration: subscriptions, usage-based overage billing |
| 27-28 | K8s deployment: Helm charts, HPA, production clusters, staging environment |
| 29-30 | Public launch: docs site, SDK publishing, marketing site, onboarding flow |

---

## Scalability Considerations

**At 1 million agents (heartbeat every 10s = 100K events/sec):**
- Ingestion layer: stateless, horizontally scalable
- NATS JetStream: 3-node cluster (tested to 10M msg/sec)
- ClickHouse: batch inserts of 10K rows/batch

**At 10 million agents (1M events/sec):**
- NATS stream sharding by tenant hash
- ClickHouse cluster sharding by tenant_id
- 50-100 ingestion pods
- Redis cluster 6+ nodes

---

## Key Design Decisions

| Decision | Rationale |
|---|---|
| LLM Proxy as MVP killer feature | Zero-code integration (env var change only) maximizes adoption; covers 4/5 domains without agent modifications |
| 2-tier integration (Proxy → SDK) | Proxy for zero-friction onboarding, SDK only for conflict prevention; OTel supported as ingestion format, not a separate tier |
| Separate ingestion and API binaries | Independent scaling; dashboard queries don't starve heartbeat processing |
| ClickHouse for both metrics and audit | Reduced operational complexity, same query patterns, splittable later |
| NATS over Kafka | Lower latency, simpler operations, swappable via adapter pattern |
| Redis distributed locks over etcd | Already in stack, sufficient for advisory locks, etcd can be added later |
| gRPC bidirectional streaming | Enables server-to-agent push, natural backpressure, no polling needed |

---

## Getting Started

```bash
# Initialize the project
go mod init github.com/YoungsoonLee/meowsight

# Start local infrastructure
docker-compose up -d  # PostgreSQL, ClickHouse, Redis, NATS

# Build
make build

# Test
make test

# Run API server
./bin/meowsight-api

# Run LLM proxy
./bin/meowsight-proxy

# Run ingestion worker
./bin/meowsight-ingest
```

---

## License

Proprietary - All rights reserved
