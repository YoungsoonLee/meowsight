package nats

import (
	"testing"
)

func TestConsumerFilterSubject(t *testing.T) {
	expected := "events.*.request"
	actual := subjectPrefix + ".*.request"
	if actual != expected {
		t.Errorf("expected filter subject %s, got %s", expected, actual)
	}
}

func TestConsumerName(t *testing.T) {
	names := []string{"metric-writer", "audit-writer", "cost-aggregator"}
	for _, name := range names {
		if name == "" {
			t.Error("consumer name must not be empty")
		}
	}
}
