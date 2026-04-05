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

// EventHandler processes a single RequestEvent.
type EventHandler func(ctx context.Context, event proxy.RequestEvent) error

// Consumer subscribes to the EVENTS stream and dispatches events to handlers.
type Consumer struct {
	js       jetstream.JetStream
	consumer jetstream.Consumer
	handlers []EventHandler
	stop     context.CancelFunc
}

// NewConsumer creates a durable JetStream consumer for processing request events.
func NewConsumer(ctx context.Context, natsURL, consumerName string, handlers ...EventHandler) (*Consumer, error) {
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

	cons, err := js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: subjectPrefix + ".*.request",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    5,
	})
	if err != nil {
		return nil, fmt.Errorf("jetstream create consumer: %w", err)
	}

	slog.Info("nats consumer ready", "consumer", consumerName, "stream", streamName)
	return &Consumer{
		js:       js,
		consumer: cons,
		handlers: handlers,
	}, nil
}

// Start begins consuming messages. Blocks until ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.stop = cancel

	iter, err := c.consumer.Messages(jetstream.PullMaxMessages(10))
	if err != nil {
		cancel()
		return fmt.Errorf("consumer messages: %w", err)
	}

	slog.Info("nats consumer started, waiting for events")

	for {
		select {
		case <-ctx.Done():
			iter.Stop()
			return ctx.Err()
		default:
		}

		msg, err := iter.Next()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Error("consumer next error", "error", err)
			continue
		}

		var event proxy.RequestEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			slog.Error("failed to unmarshal event", "error", err, "subject", msg.Subject())
			msg.Term()
			continue
		}

		allOK := true
		for _, h := range c.handlers {
			if err := h(ctx, event); err != nil {
				slog.Error("handler error",
					"error", err,
					"tenant_id", event.TenantID,
					"model", event.Model,
				)
				allOK = false
			}
		}

		if allOK {
			msg.Ack()
		} else {
			msg.Nak()
		}
	}
}

// Stop cancels the consumer loop.
func (c *Consumer) Stop() {
	if c.stop != nil {
		c.stop()
	}
}
