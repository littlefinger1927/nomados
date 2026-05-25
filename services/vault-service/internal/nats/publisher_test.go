package nats

import (
	"encoding/json"
	"testing"
)

func TestKeyRotatedEventMarshaling(t *testing.T) {
	t.Run("event marshals and unmarshals correctly", func(t *testing.T) {
		event := KeyRotatedEvent{
			WorkspaceID: "ws-123",
			KeyID:       "key-456",
		}

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal event: %v", err)
		}

		var unmarshaled KeyRotatedEvent
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}

		if unmarshaled.WorkspaceID != event.WorkspaceID {
			t.Errorf("expected workspace ID %s, got %s", event.WorkspaceID, unmarshaled.WorkspaceID)
		}
		if unmarshaled.KeyID != event.KeyID {
			t.Errorf("expected key ID %s, got %s", event.KeyID, unmarshaled.KeyID)
		}
	})
}