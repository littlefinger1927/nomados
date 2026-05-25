package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// SessionRevokedEvent represents a session revocation event.
type SessionRevokedEvent struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	RevokedAt time.Time `json:"revoked_at"`
}

// AllSessionsRevokedEvent represents a bulk session revocation event.
type AllSessionsRevokedEvent struct {
	UserID    string    `json:"user_id"`
	RevokedAt time.Time `json:"revoked_at"`
}

// Publisher publishes session events to NATS.
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

// PublishSessionRevoked publishes a session.revoked event.
func (p *Publisher) PublishSessionRevoked(ctx context.Context, sessionID, userID string) error {
	event := SessionRevokedEvent{
		SessionID: sessionID,
		UserID:    userID,
		RevokedAt:  time.Now().UTC(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal session revoked event: %w", err)
	}

	if err := p.conn.Publish("sessions.revoked", data); err != nil {
		return fmt.Errorf("failed to publish session revoked event: %w", err)
	}
	return nil
}

// PublishAllSessionsRevoked publishes a sessions.all_revoked event.
func (p *Publisher) PublishAllSessionsRevoked(ctx context.Context, userID string) error {
	event := AllSessionsRevokedEvent{
		UserID:    userID,
		RevokedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal all sessions revoked event: %w", err)
	}

	if err := p.conn.Publish("sessions.all_revoked", data); err != nil {
		return fmt.Errorf("failed to publish all sessions revoked event: %w", err)
	}
	return nil
}

// Close closes the NATS connection.
func (p *Publisher) Close() {
	if p.conn != nil {
		p.conn.Close()
	}
}