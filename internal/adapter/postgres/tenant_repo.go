package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tenant represents a row in the tenants table.
type Tenant struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Plan       string    `json:"plan"`
	APIKey     string    `json:"api_key,omitempty"` // only populated on create
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TenantRepo manages tenants in PostgreSQL.
type TenantRepo struct {
	pool *pgxpool.Pool
}

// NewTenantRepo creates a TenantRepo sharing the given pool.
func NewTenantRepo(pool *pgxpool.Pool) *TenantRepo {
	return &TenantRepo{pool: pool}
}

// Create inserts a new tenant, generates an API key, and returns the tenant
// with the plaintext key (only available at creation time).
func (r *TenantRepo) Create(ctx context.Context, name, plan string) (*Tenant, error) {
	if plan == "" {
		plan = "free"
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}
	keyHash := hashKey(apiKey)

	var t Tenant
	err = r.pool.QueryRow(ctx, `
		INSERT INTO tenants (name, plan, api_key_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, plan, created_at, updated_at
	`, name, plan, keyHash).Scan(&t.ID, &t.Name, &t.Plan, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert tenant: %w", err)
	}
	t.APIKey = apiKey
	return &t, nil
}

// GetByID returns a single tenant by UUID.
func (r *TenantRepo) GetByID(ctx context.Context, id string) (*Tenant, error) {
	var t Tenant
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, plan, created_at, updated_at
		FROM tenants WHERE id = $1
	`, id).Scan(&t.ID, &t.Name, &t.Plan, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	return &t, nil
}

// List returns all tenants ordered by creation time.
func (r *TenantRepo) List(ctx context.Context) ([]Tenant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, plan, created_at, updated_at
		FROM tenants ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Plan, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

// Update modifies the name and/or plan of an existing tenant.
func (r *TenantRepo) Update(ctx context.Context, id, name, plan string) (*Tenant, error) {
	var t Tenant
	err := r.pool.QueryRow(ctx, `
		UPDATE tenants SET name = $2, plan = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, name, plan, created_at, updated_at
	`, id, name, plan).Scan(&t.ID, &t.Name, &t.Plan, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("update tenant: %w", err)
	}
	return &t, nil
}

// Delete removes a tenant by UUID.
func (r *TenantRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}

// RotateAPIKey generates a new API key for an existing tenant.
// Returns the new plaintext key (only available at rotation time).
func (r *TenantRepo) RotateAPIKey(ctx context.Context, id string) (string, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}
	keyHash := hashKey(apiKey)

	tag, err := r.pool.Exec(ctx, `
		UPDATE tenants SET api_key_hash = $2, updated_at = now()
		WHERE id = $1
	`, id, keyHash)
	if err != nil {
		return "", fmt.Errorf("rotate api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return "", fmt.Errorf("tenant not found")
	}
	return apiKey, nil
}

// Pool exposes the underlying connection pool so other repos can share it.
func (r *TenantRepo) Pool() *pgxpool.Pool {
	return r.pool
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "mst-" + hex.EncodeToString(b), nil
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
