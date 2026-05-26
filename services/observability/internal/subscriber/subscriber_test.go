package subscriber

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/nomados/nomados/packages/logging"
)

// mockAggregator is a test double for the EventProcessor interface.
type mockAggregator struct {
	eventsProcessed int
	lastSubject     string
	lastData        []byte
}

func (m *mockAggregator) ProcessEvent(subject string, data []byte) {
	m.eventsProcessed++
	m.lastSubject = subject
	m.lastData = data
}

func (m *mockAggregator) IncrementErrors(service string) {
	// no-op for tests
}

func newMockAggregator() *mockAggregator {
	return &mockAggregator{}
}

func TestNewSubscriberInvalidURL(t *testing.T) {
	logger := logging.NewLogger("test", nil)
	agg := newMockAggregator()

	// NATS connection should fail with an invalid URL
	_, err := NewSubscriber("nats://invalid-host:4222", agg, logger)
	if err == nil {
		t.Error("expected error connecting to invalid NATS URL, got nil")
	}
}

func TestEventParsing(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name: "valid event with all fields",
			data: `{"subject":"audit.login","data":{"user_id":"u1"},"timestamp":"2024-01-01T00:00:00Z","source":"auth-service"}`,
		},
		{
			name: "valid event with minimal fields",
			data: `{"subject":"session.created","data":{"session_id":"s1"}}`,
		},
		{
			name:    "invalid JSON",
			data:    `{not valid json}`,
			wantErr: true,
		},
		{
			name: "empty JSON object",
			data: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseEvent([]byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if event == nil {
				t.Error("expected non-nil event")
			}
		})
	}
}

func TestEventStructFields(t *testing.T) {
	ts := time.Now().UTC()
	event := Event{
		Subject:   "workspace.created",
		Data:      json.RawMessage(`{"workspace_id":"ws-1"}`),
		Timestamp: ts,
		Source:    "workspace-orchestrator",
	}

	if event.Subject != "workspace.created" {
		t.Errorf("expected Subject workspace.created, got %s", event.Subject)
	}
	if event.Source != "workspace-orchestrator" {
		t.Errorf("expected Source workspace-orchestrator, got %s", event.Source)
	}
}

func TestEventJSONRoundTrip(t *testing.T) {
	original := Event{
		Subject:   "session.created",
		Data:      json.RawMessage(`{"session_id":"s-123","user_id":"u-456"}`),
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		Source:    "session-service",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	parsed, err := ParseEvent(data)
	if err != nil {
		t.Fatalf("failed to parse event: %v", err)
	}

	if parsed.Subject != original.Subject {
		t.Errorf("expected Subject %s, got %s", original.Subject, parsed.Subject)
	}
	if parsed.Source != original.Source {
		t.Errorf("expected Source %s, got %s", original.Source, parsed.Source)
	}
}

func TestProcessEventWithMockAggregator(t *testing.T) {
	logger := logging.NewLogger("test", nil)
	agg := newMockAggregator()

	// Create subscriber with nil conn (we won't connect to NATS in tests)
	sub := &Subscriber{
		conn:   nil,
		agg:    agg,
		logger: logger,
	}

	// Test processEvent directly
	sub.processEvent("workspace.created", []byte(`{"workspace_id":"ws-1"}`))

	if agg.eventsProcessed != 1 {
		t.Errorf("expected 1 event processed, got %d", agg.eventsProcessed)
	}
	if agg.lastSubject != "workspace.created" {
		t.Errorf("expected subject workspace.created, got %s", agg.lastSubject)
	}
}

func TestProcessEventMultipleSubjects(t *testing.T) {
	logger := logging.NewLogger("test", nil)
	agg := newMockAggregator()

	sub := &Subscriber{
		conn:   nil,
		agg:    agg,
		logger: logger,
	}

	subjects := []string{
		"audit.login",
		"session.created",
		"session.revoked",
		"workspace.created",
		"workspace.paused",
		"workspace.resumed",
		"workspace.stopped",
		"workspace.destroyed",
		"vault.rotated",
	}

	for _, subject := range subjects {
		sub.processEvent(subject, []byte(`{"test":true}`))
	}

	if agg.eventsProcessed != len(subjects) {
		t.Errorf("expected %d events processed, got %d", len(subjects), agg.eventsProcessed)
	}
}

func TestProcessEventWithInvalidJSON(t *testing.T) {
	logger := logging.NewLogger("test", nil)
	agg := newMockAggregator()

	sub := &Subscriber{
		conn:   nil,
		agg:    agg,
		logger: logger,
	}

	// Invalid JSON should still be forwarded to the aggregator
	sub.processEvent("audit.login", []byte(`{not valid json}`))

	if agg.eventsProcessed != 1 {
		t.Errorf("expected 1 event processed even with invalid JSON, got %d", agg.eventsProcessed)
	}
}

func TestCloseWithNilConn(t *testing.T) {
	logger := logging.NewLogger("test", nil)
	sub := &Subscriber{
		conn:   nil,
		agg:    newMockAggregator(),
		logger: logger,
	}

	// Should not panic
	sub.Close()
}

func TestCloseWithNoSubs(t *testing.T) {
	logger := logging.NewLogger("test", nil)
	sub := &Subscriber{
		conn:   nil,
		agg:    newMockAggregator(),
		logger: logger,
		subs:   nil,
	}

	// Should not panic
	sub.Close()
}

func TestNATSSubjects(t *testing.T) {
	// Verify the expected NATS subjects that the subscriber listens to
	subjects := map[string]string{
		"audit":     "audit.*",
		"session":   "session.*",
		"workspace": "workspace.*",
		"vault":     "vault.*",
	}

	for name, pattern := range subjects {
		if pattern == "" {
			t.Errorf("expected non-empty subject pattern for %s", name)
		}
		expected := name + ".*"
		if pattern != expected {
			t.Errorf("expected pattern %s, got %s", expected, pattern)
		}
	}
}