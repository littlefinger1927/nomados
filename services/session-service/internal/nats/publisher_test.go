package nats

import (
	"testing"
)

func TestNewPublisherInvalidURL(t *testing.T) {
	// NATS connection should fail with an invalid URL
	_, err := NewPublisher("nats://invalid-host:4222")
	if err == nil {
		t.Error("expected error connecting to invalid NATS URL, got nil")
	}
}

func TestSessionRevokedEventMarshal(t *testing.T) {
	event := SessionRevokedEvent{
		SessionID: "session-123",
		UserID:    "user-456",
	}
	if event.SessionID != "session-123" {
		t.Errorf("expected SessionID session-123, got %s", event.SessionID)
	}
	if event.UserID != "user-456" {
		t.Errorf("expected UserID user-456, got %s", event.UserID)
	}
}

func TestAllSessionsRevokedEventMarshal(t *testing.T) {
	event := AllSessionsRevokedEvent{
		UserID: "user-789",
	}
	if event.UserID != "user-789" {
		t.Errorf("expected UserID user-789, got %s", event.UserID)
	}
}

func TestPublisherCloseNilConn(t *testing.T) {
	// Closing a publisher with nil connection should not panic
	p := &Publisher{conn: nil}
	p.Close() // should not panic
}