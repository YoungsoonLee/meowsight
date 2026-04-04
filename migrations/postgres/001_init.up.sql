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
CREATE TABLE IF NOT EXISTS agents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        TEXT NOT NULL,
    group_name  TEXT NOT NULL DEFAULT 'default',
    status      TEXT NOT NULL DEFAULT 'offline',
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_agents_tenant_id ON agents(tenant_id);
CREATE INDEX idx_agents_status ON agents(tenant_id, status);

-- Model pricing
CREATE TABLE IF NOT EXISTS model_pricing (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider        TEXT NOT NULL,
    model           TEXT NOT NULL,
    input_price_per_1k  NUMERIC(10, 6) NOT NULL,
    output_price_per_1k NUMERIC(10, 6) NOT NULL,
    effective_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, model, effective_date)
);

-- Budgets
CREATE TABLE IF NOT EXISTS budgets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    scope       TEXT NOT NULL DEFAULT 'tenant',
    scope_id    UUID,
    period      TEXT NOT NULL DEFAULT 'monthly',
    limit_usd   NUMERIC(12, 4) NOT NULL,
    current_spend_usd NUMERIC(12, 4) NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_budgets_tenant_id ON budgets(tenant_id);

-- Audit chain hashes (tamper evidence)
CREATE TABLE IF NOT EXISTS audit_chain (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    batch_hash  TEXT NOT NULL,
    prev_hash   TEXT NOT NULL,
    record_count INT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_chain_tenant ON audit_chain(tenant_id);
