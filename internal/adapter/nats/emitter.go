package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/YoungsoonLee/meowsight/internal/proxy"
)

const (
	streamName = "EVENTS"
	// Subject pattern: events.{tenant_id}.request
	subjectPrefix = "events"
)

// Emitter publishes proxy RequestEvents to NATS JetStream.
type Emitter struct {
	js jetstream.JetStream
}

// NewEmitter connects to NATS JetStream and ensures the EVENTS stream exists.
func NewEmitter(ctx context.Context, natsURL string) (*Emitter, error) {
	nc, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("jetstream new: %w", err)
	}

	// Create or update the EVENTS stream.
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{subjectPrefix + ".>"},
		Retention: jetstream.WorkQueuePolicy,
		MaxAge:    72 * time.Hour, // 72h replay window
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		return nil, fmt.Errorf("jetstream create stream: %w", err)
	}

	slog.Info("nats jetstream emitter ready", "stream", streamName, "url", natsURL)
	return &Emitter{js: js}, nil
}

// Emit publishes a RequestEvent to JetStream.
// Subject: events.{tenant_id}.request
func (e *Emitter) Emit(event proxy.RequestEvent) {
	tenantID := event.TenantID
	if tenantID == "" {
		tenantID = "default"
	}

	subject := fmt.Sprintf("%s.%s.request", subjectPrefix, tenantID)

	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("failed to marshal event", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = e.js.Publish(ctx, subject, data)
	if err != nil {
		slog.Error("failed to publish event to jetstream",
			"subject", subject,
			"error", err,
		)
		return
	}

	slog.Debug("event published to jetstream", "subject", subject, "model", event.Model)
}
