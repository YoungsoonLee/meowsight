package proxy

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// KeyMapping holds the resolved identity and upstream key for a MeowSight API key.
type KeyMapping struct {
	TenantID       string
	AgentID        string
	Provider       string
	UpstreamAPIKey string
}

// KeyStore is the interface for looking up API key mappings.
type KeyStore interface {
	LookupByHash(ctx context.Context, keyHash string) (*KeyMapping, error)
}

// KeyResolver resolves MeowSight API keys to tenant/agent identity.
// It caches lookups in memory to avoid hitting the database on every request.
type KeyResolver struct {
	store KeyStore
	cache sync.Map // keyHash → *cachedEntry
	ttl   time.Duration
}

type cachedEntry struct {
	mapping   *KeyMapping
	expiresAt time.Time
}

// NewKeyResolver creates a key resolver with the given store and cache TTL.
func NewKeyResolver(store KeyStore, cacheTTL time.Duration) *KeyResolver {
	return &KeyResolver{
		store: store,
		ttl:   cacheTTL,
	}
}

// Resolve looks up a MeowSight API key and returns the mapping.
// Returns nil if the key is not found or inactive.
func (kr *KeyResolver) Resolve(ctx context.Context, apiKey string) *KeyMapping {
	hash := HashKey(apiKey)

	// Check cache first
	if entry, ok := kr.cache.Load(hash); ok {
		ce := entry.(*cachedEntry)
		if time.Now().Before(ce.expiresAt) {
			return ce.mapping
		}
		kr.cache.Delete(hash)
	}

	// Lookup in store
	mapping, err := kr.store.LookupByHash(ctx, hash)
	if err != nil {
		slog.Error("key resolver lookup failed", "error", err)
		return nil
	}

	// Cache the result (even nil, to avoid repeated DB misses)
	kr.cache.Store(hash, &cachedEntry{
		mapping:   mapping,
		expiresAt: time.Now().Add(kr.ttl),
	})

	return mapping
}

// HashKey returns the SHA-256 hex hash of an API key.
func HashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h)
}

// KeyPrefix returns the first 8 characters of a key for quick identification.
func KeyPrefix(key string) string {
	if len(key) <= 8 {
		return key
	}
	return key[:8]
}
