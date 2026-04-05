package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YoungsoonLee/meowsight/internal/proxy"
)

// KeyStore implements proxy.KeyStore using PostgreSQL.
type KeyStore struct {
	pool *pgxpool.Pool
}

// NewKeyStore creates a key store backed by the api_keys table.
func NewKeyStore(pool *pgxpool.Pool) *KeyStore {
	return &KeyStore{pool: pool}
}

// LookupByHash finds an active API key mapping by its SHA-256 hash.
func (s *KeyStore) LookupByHash(ctx context.Context, keyHash string) (*proxy.KeyMapping, error) {
	var m proxy.KeyMapping
	err := s.pool.QueryRow(ctx, `
		SELECT tenant_id, agent_id, provider, upstream_api_key
		FROM api_keys
		WHERE key_hash = $1 AND active = true
	`, keyHash).Scan(&m.TenantID, &m.AgentID, &m.Provider, &m.UpstreamAPIKey)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("lookup api key: %w", err)
	}
	return &m, nil
}
