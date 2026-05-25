package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/streaming-service/internal/relay"

	natsgo "github.com/nats-io/nats.go"
)

// WorkspaceCreatedEvent represents a workspace creation event.
type WorkspaceCreatedEvent struct {
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// WorkspaceStoppedEvent represents a workspace stop event.
type WorkspaceStoppedEvent struct {
	WorkspaceID string    `json:"workspace_id"`
	StoppedAt   time.Time `json:"stopped_at"`
}

// WorkspaceDestroyedEvent represents a workspace destruction event.
type WorkspaceDestroyedEvent struct {
	WorkspaceID string    `json:"workspace_id"`
	DestroyedAt time.Time `json:"destroyed_at"`
}

// Subscriber subscribes to workspace events via NATS and manages
// stream relay lifecycle accordingly.
type Subscriber struct {
	conn   *natsgo.Conn
	relay  *relay.StreamRelay
	logger *logging.Logger
	subs   []*natsgo.Subscription
}

// NewSubscriber connects to NATS and returns a new Subscriber.
func NewSubscriber(natsURL string, streamRelay *relay.StreamRelay, logger *logging.Logger) (*Subscriber, error) {
	conn, err := natsgo.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &Subscriber{
		conn:   conn,
		relay:  streamRelay,
		logger: logger,
	}, nil
}

// SubscribeAll sets up subscriptions for all workspace events that
// affect stream lifecycle.
func (s *Subscriber) SubscribeAll(ctx context.Context) error {
	// Subscribe to workspace.created events — prepare a stream
	createdSub, err := s.conn.Subscribe("workspace.created", func(msg *natsgo.Msg) {
		var event WorkspaceCreatedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			s.logger.Error("failed to unmarshal workspace.created event", "error", err)
			return
		}

		s.logger.Info("received workspace.created event", "workspace_id", event.WorkspaceID, "user_id", event.UserID)

		if _, err := s.relay.CreateStream(ctx, event.WorkspaceID); err != nil {
			s.logger.Error("failed to create stream for workspace", "workspace_id", event.WorkspaceID, "error", err)
			return
		}

		s.logger.Info("stream created for workspace", "workspace_id", event.WorkspaceID)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to workspace.created: %w", err)
	}
	s.subs = append(s.subs, createdSub)

	// Subscribe to workspace.stopped events — close the stream
	stoppedSub, err := s.conn.Subscribe("workspace.stopped", func(msg *natsgo.Msg) {
		var event WorkspaceStoppedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			s.logger.Error("failed to unmarshal workspace.stopped event", "error", err)
			return
		}

		s.logger.Info("received workspace.stopped event", "workspace_id", event.WorkspaceID)

		if err := s.relay.CloseStream(ctx, event.WorkspaceID); err != nil {
			s.logger.Error("failed to close stream for workspace", "workspace_id", event.WorkspaceID, "error", err)
			return
		}

		s.logger.Info("stream closed for workspace", "workspace_id", event.WorkspaceID)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to workspace.stopped: %w", err)
	}
	s.subs = append(s.subs, stoppedSub)

	// Subscribe to workspace.destroyed events — close the stream if running
	destroyedSub, err := s.conn.Subscribe("workspace.destroyed", func(msg *natsgo.Msg) {
		var event WorkspaceDestroyedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			s.logger.Error("failed to unmarshal workspace.destroyed event", "error", err)
			return
		}

		s.logger.Info("received workspace.destroyed event", "workspace_id", event.WorkspaceID)

		if err := s.relay.CloseStream(ctx, event.WorkspaceID); err != nil {
			s.logger.Warn("failed to close stream during workspace destruction", "workspace_id", event.WorkspaceID, "error", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to workspace.destroyed: %w", err)
	}
	s.subs = append(s.subs, destroyedSub)

	s.logger.Info("subscribed to workspace events", "subjects", "workspace.created, workspace.stopped, workspace.destroyed")

	return nil
}

// Close unsubscribes from all subjects and closes the NATS connection.
func (s *Subscriber) Close() {
	for _, sub := range s.subs {
		if err := sub.Unsubscribe(); err != nil {
			s.logger.Warn("failed to unsubscribe", "error", err)
		}
	}
	s.subs = nil

	if s.conn != nil {
		s.conn.Close()
	}
}