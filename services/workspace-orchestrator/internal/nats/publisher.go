package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// WorkspaceCreatedEvent represents a workspace creation event.
type WorkspaceCreatedEvent struct {
	WorkspaceID string    `json:"workspace_id"`
	UserID     string    `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// WorkspacePausedEvent represents a workspace pause event.
type WorkspacePausedEvent struct {
	WorkspaceID string    `json:"workspace_id"`
	PausedAt   time.Time `json:"paused_at"`
}

// WorkspaceResumedEvent represents a workspace resume event.
type WorkspaceResumedEvent struct {
	WorkspaceID string    `json:"workspace_id"`
	ResumedAt  time.Time `json:"resumed_at"`
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

// Publisher publishes workspace events to NATS.
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

// PublishWorkspaceCreated publishes a workspace.created event.
func (p *Publisher) PublishWorkspaceCreated(ctx context.Context, workspaceID, userID string) error {
	event := WorkspaceCreatedEvent{
		WorkspaceID: workspaceID,
		UserID:     userID,
		CreatedAt:  time.Now().UTC(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace created event: %w", err)
	}

	if err := p.conn.Publish("workspace.created", data); err != nil {
		return fmt.Errorf("failed to publish workspace created event: %w", err)
	}
	return nil
}

// PublishWorkspacePaused publishes a workspace.paused event.
func (p *Publisher) PublishWorkspacePaused(ctx context.Context, workspaceID string) error {
	event := WorkspacePausedEvent{
		WorkspaceID: workspaceID,
		PausedAt:   time.Now().UTC(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace paused event: %w", err)
	}

	if err := p.conn.Publish("workspace.paused", data); err != nil {
		return fmt.Errorf("failed to publish workspace paused event: %w", err)
	}
	return nil
}

// PublishWorkspaceResumed publishes a workspace.resumed event.
func (p *Publisher) PublishWorkspaceResumed(ctx context.Context, workspaceID string) error {
	event := WorkspaceResumedEvent{
		WorkspaceID: workspaceID,
		ResumedAt:  time.Now().UTC(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace resumed event: %w", err)
	}

	if err := p.conn.Publish("workspace.resumed", data); err != nil {
		return fmt.Errorf("failed to publish workspace resumed event: %w", err)
	}
	return nil
}

// PublishWorkspaceStopped publishes a workspace.stopped event.
func (p *Publisher) PublishWorkspaceStopped(ctx context.Context, workspaceID string) error {
	event := WorkspaceStoppedEvent{
		WorkspaceID: workspaceID,
		StoppedAt:   time.Now().UTC(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace stopped event: %w", err)
	}

	if err := p.conn.Publish("workspace.stopped", data); err != nil {
		return fmt.Errorf("failed to publish workspace stopped event: %w", err)
	}
	return nil
}

// PublishWorkspaceDestroyed publishes a workspace.destroyed event.
func (p *Publisher) PublishWorkspaceDestroyed(ctx context.Context, workspaceID string) error {
	event := WorkspaceDestroyedEvent{
		WorkspaceID: workspaceID,
		DestroyedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace destroyed event: %w", err)
	}

	if err := p.conn.Publish("workspace.destroyed", data); err != nil {
		return fmt.Errorf("failed to publish workspace destroyed event: %w", err)
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