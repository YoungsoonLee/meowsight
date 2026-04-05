package proxy

import (
	"context"
	"testing"
	"time"
)

// mockKeyStore implements KeyStore for testing.
type mockKeyStore struct {
	keys map[string]*KeyMapping
}

func (s *mockKeyStore) LookupByHash(_ context.Context, keyHash string) (*KeyMapping, error) {
	m, ok := s.keys[keyHash]
	if !ok {
		return nil, nil
	}
	return m, nil
}

func TestKeyResolver_Resolve(t *testing.T) {
	store := &mockKeyStore{
		keys: map[string]*KeyMapping{
			HashKey("ms-test-key-123"): {
				TenantID:       "tenant-1",
				AgentID:        "cursor-agent",
				Provider:       "openai",
				UpstreamAPIKey: "sk-real-openai-key",
			},
		},
	}

	resolver := NewKeyResolver(store, 5*time.Minute)

	// Known key
	mapping := resolver.Resolve(context.Background(), "ms-test-key-123")
	if mapping == nil {
		t.Fatal("expected mapping for known key")
	}
	if mapping.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", mapping.TenantID)
	}
	if mapping.AgentID != "cursor-agent" {
		t.Errorf("expected cursor-agent, got %s", mapping.AgentID)
	}
	if mapping.UpstreamAPIKey != "sk-real-openai-key" {
		t.Errorf("expected sk-real-openai-key, got %s", mapping.UpstreamAPIKey)
	}

	// Unknown key
	mapping = resolver.Resolve(context.Background(), "ms-unknown-key")
	if mapping != nil {
		t.Error("expected nil for unknown key")
	}
}

func TestKeyResolver_Cache(t *testing.T) {
	callCount := 0
	store := &mockKeyStore{
		keys: map[string]*KeyMapping{
			HashKey("ms-cached-key"): {TenantID: "t-1", AgentID: "a-1", Provider: "openai", UpstreamAPIKey: "sk-real"},
		},
	}
	// Wrap to count calls
	countingStore := &countingKeyStore{store: store, count: &callCount}

	resolver := NewKeyResolver(countingStore, 5*time.Minute)

	// First call — hits store
	resolver.Resolve(context.Background(), "ms-cached-key")
	if callCount != 1 {
		t.Errorf("expected 1 store call, got %d", callCount)
	}

	// Second call — should hit cache
	resolver.Resolve(context.Background(), "ms-cached-key")
	if callCount != 1 {
		t.Errorf("expected 1 store call (cached), got %d", callCount)
	}
}

type countingKeyStore struct {
	store *mockKeyStore
	count *int
}

func (s *countingKeyStore) LookupByHash(ctx context.Context, keyHash string) (*KeyMapping, error) {
	*s.count++
	return s.store.LookupByHash(ctx, keyHash)
}

func TestHashKey(t *testing.T) {
	hash := HashKey("test-key")
	if len(hash) != 64 { // SHA-256 hex = 64 chars
		t.Errorf("expected 64 char hash, got %d", len(hash))
	}

	// Same input = same hash
	if HashKey("test-key") != hash {
		t.Error("expected deterministic hash")
	}

	// Different input = different hash
	if HashKey("other-key") == hash {
		t.Error("expected different hash for different input")
	}
}

func TestKeyPrefix(t *testing.T) {
	if KeyPrefix("ms-test-key-12345") != "ms-test-" {
		t.Errorf("expected ms-test-, got %s", KeyPrefix("ms-test-key-12345"))
	}
	if KeyPrefix("short") != "short" {
		t.Errorf("expected short, got %s", KeyPrefix("short"))
	}
}
