-- ============================================================
-- MeowSight PostgreSQL Schema
-- ============================================================

-- Tenants
CREATE TABLE IF NOT EXISTS tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    plan        TEXT NOT NULL DEFAULT 'free',
    api_key_hash TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Agents
-- Supports both auto-discovery (external_tenant_id + external_agent_id from proxy headers)
-- and managed registration (tenant_id FK linked after tenant signs up).
CREATE TABLE IF NOT EXISTS agents (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID REFERENCES tenants(id),   -- nullable: linked after tenant registration
    external_tenant_id  TEXT NOT NULL DEFAULT 'default',
    external_agent_id   TEXT NOT NULL DEFAULT 'unknown',
    name                TEXT NOT NULL DEFAULT '',
    group_name          TEXT NOT NULL DEFAULT 'default',
    status              TEXT NOT NULL DEFAULT 'active',
    provider            TEXT NOT NULL DEFAULT '',
    model               TEXT NOT NULL DEFAULT '',
    first_seen_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    request_count       BIGINT NOT NULL DEFAULT 0,
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(external_tenant_id, external_agent_id)
);

CREATE INDEX idx_agents_tenant_id ON agents(tenant_id);
CREATE INDEX idx_agents_status ON agents(tenant_id, status);
CREATE INDEX idx_agents_external_tenant ON agents(external_tenant_id, last_seen_at);

-- Model pricing
CREATE TABLE IF NOT EXISTS model_pricing (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider            TEXT NOT NULL,
    model               TEXT NOT NULL,
    input_price_per_1k  NUMERIC(10, 6) NOT NULL,
    output_price_per_1k NUMERIC(10, 6) NOT NULL,
    effective_date      DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, model, effective_date)
);

-- Budgets
CREATE TABLE IF NOT EXISTS budgets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    scope             TEXT NOT NULL DEFAULT 'tenant',
    scope_id          UUID,
    period            TEXT NOT NULL DEFAULT 'monthly',
    limit_usd         NUMERIC(12, 4) NOT NULL,
    current_spend_usd NUMERIC(12, 4) NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_budgets_tenant_id ON budgets(tenant_id);

-- Audit chain hashes (tamper evidence)
CREATE TABLE IF NOT EXISTS audit_chain (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    batch_hash   TEXT NOT NULL,
    prev_hash    TEXT NOT NULL,
    record_count INT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_chain_tenant ON audit_chain(tenant_id);

-- API keys for agent identification
-- Agents that cannot add custom headers use MeowSight-issued API keys.
-- The proxy resolves the key to tenant/agent and swaps it with the real LLM API key.
CREATE TABLE IF NOT EXISTS api_keys (
    id               BIGSERIAL PRIMARY KEY,
    key_prefix       TEXT NOT NULL,              -- first 8 chars (e.g. "ms-abc12") for quick lookup
    key_hash         TEXT NOT NULL UNIQUE,        -- SHA-256 hash of full key
    tenant_id        TEXT NOT NULL,
    agent_id         TEXT NOT NULL,
    provider         TEXT NOT NULL,               -- "openai", "anthropic"
    upstream_api_key TEXT NOT NULL,               -- real LLM provider API key (encrypted in production)
    description      TEXT NOT NULL DEFAULT '',
    active           BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_api_keys_prefix ON api_keys(key_prefix) WHERE active = true;
CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
