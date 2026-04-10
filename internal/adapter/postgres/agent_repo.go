package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Agent represents an agent record in the agents table.
type Agent struct {
	ID               string
	TenantUUID       *string // nullable FK to tenants.id
	ExternalTenantID string
	ExternalAgentID  string
	Name             string
	Status           string
	Provider         string
	Model            string
	FirstSeenAt      time.Time
	LastSeenAt       time.Time
	RequestCount     int64
}

// AgentRepo manages agents in PostgreSQL.
type AgentRepo struct {
	pool *pgxpool.Pool
}

// NewAgentRepo creates a connection pool and returns an agent repository.
func NewAgentRepo(ctx context.Context, dsn string) (*AgentRepo, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool new: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	slog.Info("postgres agent repo ready")
	return &AgentRepo{pool: pool}, nil
}

// Upsert inserts a new agent or updates tracking fields on conflict.
// Uses the agents table with external_tenant_id + external_agent_id for auto-discovery.
// If a tenant with a matching name exists, the agent is automatically linked via tenant_id FK.
func (r *AgentRepo) Upsert(ctx context.Context, tenantID, agentID, provider, model string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO agents (external_tenant_id, external_agent_id, name, provider, model, status,
		                    tenant_id, first_seen_at, last_seen_at, request_count)
		VALUES ($1, $2, $2, $3, $4, 'active',
		        (SELECT id FROM tenants WHERE name = $1 LIMIT 1),
		        now(), now(), 1)
		ON CONFLICT (external_tenant_id, external_agent_id)
		DO UPDATE SET
			provider = $3,
			model = $4,
			status = 'active',
			tenant_id = COALESCE(
				(SELECT id FROM tenants WHERE name = $1 LIMIT 1),
				agents.tenant_id
			),
			last_seen_at = now(),
			request_count = agents.request_count + 1,
			updated_at = now()
	`, tenantID, agentID, provider, model)
	if err != nil {
		return fmt.Errorf("upsert agent: %w", err)
	}
	return nil
}

// GetByTenant returns all agents for an external tenant ID, ordered by most recently seen.
func (r *AgentRepo) GetByTenant(ctx context.Context, tenantID string) ([]Agent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, external_tenant_id, external_agent_id, name, status, provider, model,
		       first_seen_at, last_seen_at, request_count
		FROM agents
		WHERE external_tenant_id = $1
		ORDER BY last_seen_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query agents: %w", err)
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.TenantUUID, &a.ExternalTenantID, &a.ExternalAgentID, &a.Name,
			&a.Status, &a.Provider, &a.Model, &a.FirstSeenAt, &a.LastSeenAt, &a.RequestCount); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// GetByID returns a single agent by UUID.
func (r *AgentRepo) GetByID(ctx context.Context, id string) (*Agent, error) {
	var a Agent
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, external_tenant_id, external_agent_id, name, status, provider, model,
		       first_seen_at, last_seen_at, request_count
		FROM agents WHERE id = $1
	`, id).Scan(&a.ID, &a.TenantUUID, &a.ExternalTenantID, &a.ExternalAgentID, &a.Name,
		&a.Status, &a.Provider, &a.Model, &a.FirstSeenAt, &a.LastSeenAt, &a.RequestCount)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent: %w", err)
	}
	return &a, nil
}

// Update modifies an agent's name, status, and tenant linkage.
func (r *AgentRepo) Update(ctx context.Context, id, name, status string, tenantID *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE agents SET name = $2, status = $3, tenant_id = $4, updated_at = now()
		WHERE id = $1
	`, id, name, status, tenantID)
	if err != nil {
		return fmt.Errorf("update agent: %w", err)
	}
	return nil
}

// Delete removes an agent by UUID.
func (r *AgentRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM agents WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}
	return nil
}

// ListAll returns all agents across all tenants, ordered by most recently seen.
func (r *AgentRepo) ListAll(ctx context.Context) ([]Agent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, external_tenant_id, external_agent_id, name, status, provider, model,
		       first_seen_at, last_seen_at, request_count
		FROM agents ORDER BY last_seen_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list all agents: %w", err)
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.TenantUUID, &a.ExternalTenantID, &a.ExternalAgentID, &a.Name,
			&a.Status, &a.Provider, &a.Model, &a.FirstSeenAt, &a.LastSeenAt, &a.RequestCount); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// Pool exposes the underlying connection pool for sharing with other repos.
func (r *AgentRepo) Pool() *pgxpool.Pool {
	return r.pool
}

// Close closes the connection pool.
func (r *AgentRepo) Close() {
	r.pool.Close()
}
