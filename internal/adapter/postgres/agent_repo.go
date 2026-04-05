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
func (r *AgentRepo) Upsert(ctx context.Context, tenantID, agentID, provider, model string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO agents (external_tenant_id, external_agent_id, name, provider, model, status, first_seen_at, last_seen_at, request_count)
		VALUES ($1, $2, $2, $3, $4, 'active', now(), now(), 1)
		ON CONFLICT (external_tenant_id, external_agent_id)
		DO UPDATE SET
			provider = $3,
			model = $4,
			status = 'active',
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
		SELECT id, external_tenant_id, external_agent_id, name, status, provider, model,
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
		if err := rows.Scan(&a.ID, &a.ExternalTenantID, &a.ExternalAgentID, &a.Name,
			&a.Status, &a.Provider, &a.Model, &a.FirstSeenAt, &a.LastSeenAt, &a.RequestCount); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// Close closes the connection pool.
func (r *AgentRepo) Close() {
	r.pool.Close()
}
