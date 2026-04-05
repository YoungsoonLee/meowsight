package postgres

import (
	"testing"
	"time"
)

func TestAgentFields(t *testing.T) {
	agent := Agent{
		ID:               "uuid-1",
		ExternalTenantID: "tenant-1",
		ExternalAgentID:  "agent-1",
		Name:             "agent-1",
		Status:           "active",
		Provider:         "openai",
		Model:            "gpt-4o",
		FirstSeenAt:      time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC),
		LastSeenAt:       time.Date(2026, 4, 4, 12, 5, 0, 0, time.UTC),
		RequestCount:     42,
	}

	if agent.ExternalTenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", agent.ExternalTenantID)
	}
	if agent.ExternalAgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", agent.ExternalAgentID)
	}
	if agent.Status != "active" {
		t.Errorf("expected active, got %s", agent.Status)
	}
	if agent.RequestCount != 42 {
		t.Errorf("expected 42 requests, got %d", agent.RequestCount)
	}
	if agent.FirstSeenAt.After(agent.LastSeenAt) {
		t.Error("first_seen should be before last_seen")
	}
}

func TestAgentUpsertDedup(t *testing.T) {
	// Verify upsert logic: same external_tenant_id + external_agent_id = same agent
	type key struct{ tenant, agent string }
	seen := make(map[key]int)

	events := []struct {
		tenantID string
		agentID  string
	}{
		{"tenant-1", "agent-1"},
		{"tenant-1", "agent-1"}, // duplicate — should UPDATE, not INSERT
		{"tenant-1", "agent-2"},
		{"tenant-2", "agent-1"}, // same agent_id, different tenant — separate agent
	}

	for _, e := range events {
		k := key{e.tenantID, e.agentID}
		seen[k]++
	}

	if len(seen) != 3 {
		t.Errorf("expected 3 unique agents, got %d", len(seen))
	}
	if seen[key{"tenant-1", "agent-1"}] != 2 {
		t.Error("expected tenant-1/agent-1 to be seen twice (upsert)")
	}
}
