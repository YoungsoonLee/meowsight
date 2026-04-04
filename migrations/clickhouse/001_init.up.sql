-- Metrics table
CREATE TABLE IF NOT EXISTS metrics (
    tenant_id   String,
    agent_id    String,
    metric_name String,
    value       Float64,
    labels      Map(String, String),
    timestamp   DateTime64(3)
) ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(timestamp))
ORDER BY (tenant_id, agent_id, metric_name, timestamp)
TTL toDateTime(timestamp) + INTERVAL 90 DAY;

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_log (
    id          String,
    tenant_id   String,
    agent_id    String,
    action      String,
    resource    String,
    provider    String,
    model       String,
    input_tokens  UInt32,
    output_tokens UInt32,
    cost_usd    Float64,
    latency_ms  UInt32,
    status_code UInt16,
    error       String DEFAULT '',
    metadata    Map(String, String),
    timestamp   DateTime64(3)
) ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(timestamp))
ORDER BY (tenant_id, agent_id, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY;
