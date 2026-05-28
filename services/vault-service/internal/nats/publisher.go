package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// KeyRotatedEvent represents a workspace key rotation event.
type KeyRotatedEvent struct {
	WorkspaceID string    `json:"workspace_id"`
	KeyID       string    `json:"key_id"`
	RotatedAt   time.Time `json:"rotated_at"`
}

// Publisher publishes vault events to NATS.
type Publisher struct {
	conn *nats.Conn
}

// NewPublisher connects to NATS and returns a new Publisher.
func NewPublisher(natsURL string) (*Publisher, error) {
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	return &Publisher{conn: conn}, nil
}

// PublishKeyRotated publishes a vault.workspace_key.rotated event.
func (p *Publisher) PublishKeyRotated(ctx context.Context, workspaceID, keyID string) error {
	event := KeyRotatedEvent{
		WorkspaceID: workspaceID,
		KeyID:       keyID,
		RotatedAt:   time.Now().UTC(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal key rotated event: %w", err)
	}

	if err := p.conn.Publish("vault.workspace_key.rotated", data); err != nil {
		return fmt.Errorf("failed to publish key rotated event: %w", err)
	}
	return nil
}

// IsConnected returns whether the NATS connection is still active.
func (p *Publisher) IsConnected() bool {
	return p.conn != nil && p.conn.IsConnected()
}

// Close closes the NATS connection.
func (p *Publisher) Close() {
	if p.conn != nil {
		p.conn.Close()
	}
}