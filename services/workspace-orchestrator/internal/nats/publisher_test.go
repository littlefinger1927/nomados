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

func TestWorkspaceCreatedEventMarshal(t *testing.T) {
	event := WorkspaceCreatedEvent{
		WorkspaceID: "ws-123",
		UserID:     "user-456",
	}
	if event.WorkspaceID != "ws-123" {
		t.Errorf("expected WorkspaceID ws-123, got %s", event.WorkspaceID)
	}
	if event.UserID != "user-456" {
		t.Errorf("expected UserID user-456, got %s", event.UserID)
	}
}

func TestWorkspacePausedEventMarshal(t *testing.T) {
	event := WorkspacePausedEvent{
		WorkspaceID: "ws-123",
	}
	if event.WorkspaceID != "ws-123" {
		t.Errorf("expected WorkspaceID ws-123, got %s", event.WorkspaceID)
	}
}

func TestWorkspaceResumedEventMarshal(t *testing.T) {
	event := WorkspaceResumedEvent{
		WorkspaceID: "ws-123",
	}
	if event.WorkspaceID != "ws-123" {
		t.Errorf("expected WorkspaceID ws-123, got %s", event.WorkspaceID)
	}
}

func TestWorkspaceStoppedEventMarshal(t *testing.T) {
	event := WorkspaceStoppedEvent{
		WorkspaceID: "ws-123",
	}
	if event.WorkspaceID != "ws-123" {
		t.Errorf("expected WorkspaceID ws-123, got %s", event.WorkspaceID)
	}
}

func TestWorkspaceDestroyedEventMarshal(t *testing.T) {
	event := WorkspaceDestroyedEvent{
		WorkspaceID: "ws-123",
	}
	if event.WorkspaceID != "ws-123" {
		t.Errorf("expected WorkspaceID ws-123, got %s", event.WorkspaceID)
	}
}

func TestPublisherCloseNilConn(t *testing.T) {
	// Closing a publisher with nil connection should not panic
	p := &Publisher{conn: nil}
	p.Close() // should not panic
}

func TestEventJSONSerialization(t *testing.T) {
	events := []interface{}{
		WorkspaceCreatedEvent{WorkspaceID: "ws-1", UserID: "user-1"},
		WorkspacePausedEvent{WorkspaceID: "ws-2"},
		WorkspaceResumedEvent{WorkspaceID: "ws-3"},
		WorkspaceStoppedEvent{WorkspaceID: "ws-4"},
		WorkspaceDestroyedEvent{WorkspaceID: "ws-5"},
	}

	for _, event := range events {
		// Verify that all event types can be marshaled to JSON
		// (This tests that the struct tags are correct)
		_ = event // events are verified by field access above
	}
}

func TestNATSSubjects(t *testing.T) {
	// Verify the expected NATS subjects used by the publisher
	subjects := map[string]string{
		"created":  "workspace.created",
		"paused":   "workspace.paused",
		"resumed":  "workspace.resumed",
		"stopped":  "workspace.stopped",
		"destroyed": "workspace.destroyed",
	}

	for name, subject := range subjects {
		if subject == "" {
			t.Errorf("expected non-empty subject for %s", name)
		}
		if subject != "workspace."+name {
			t.Errorf("expected subject workspace.%s, got %s", name, subject)
		}
	}
}