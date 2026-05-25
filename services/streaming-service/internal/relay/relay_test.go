package relay

import (
	"context"
	"testing"

	"github.com/nomados/nomados/packages/logging"
)

func newTestRelay() *StreamRelay {
	logger := logging.NewLogger("streaming-test", nil)
	return NewStreamRelay(logger)
}

func TestCreateStream(t *testing.T) {
	relay := newTestRelay()
	ctx := context.Background()

	stream, err := relay.CreateStream(ctx, "ws-1")
	if err != nil {
		t.Fatalf("CreateStream returned error: %v", err)
	}

	if stream.WorkspaceID != "ws-1" {
		t.Errorf("expected WorkspaceID ws-1, got %s", stream.WorkspaceID)
	}
	if stream.Status != StreamActive {
		t.Errorf("expected status active, got %s", stream.Status)
	}
	if stream.ID == "" {
		t.Error("expected non-empty stream ID")
	}
}

func TestCreateStreamDuplicate(t *testing.T) {
	relay := newTestRelay()
	ctx := context.Background()

	_, err := relay.CreateStream(ctx, "ws-1")
	if err != nil {
		t.Fatalf("first CreateStream returned error: %v", err)
	}

	_, err = relay.CreateStream(ctx, "ws-1")
	if err == nil {
		t.Error("expected error for duplicate stream, got nil")
	}
}

func TestGetStream(t *testing.T) {
	relay := newTestRelay()
	ctx := context.Background()

	created, _ := relay.CreateStream(ctx, "ws-1")
	found, err := relay.GetStream("ws-1")
	if err != nil {
		t.Fatalf("GetStream returned error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("expected stream ID %s, got %s", created.ID, found.ID)
	}
}

func TestGetStreamNotFound(t *testing.T) {
	relay := newTestRelay()

	_, err := relay.GetStream("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent stream, got nil")
	}
}

func TestCloseStream(t *testing.T) {
	relay := newTestRelay()
	ctx := context.Background()

	relay.CreateStream(ctx, "ws-1")
	err := relay.CloseStream(ctx, "ws-1")
	if err != nil {
		t.Fatalf("CloseStream returned error: %v", err)
	}

	_, err = relay.GetStream("ws-1")
	if err == nil {
		t.Error("expected error getting closed stream, got nil")
	}
}

func TestCloseStreamNotFound(t *testing.T) {
	relay := newTestRelay()
	ctx := context.Background()

	err := relay.CloseStream(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error closing nonexistent stream, got nil")
	}
}

func TestListStreams(t *testing.T) {
	relay := newTestRelay()
	ctx := context.Background()

	relay.CreateStream(ctx, "ws-1")
	relay.CreateStream(ctx, "ws-2")
	relay.CreateStream(ctx, "ws-3")

	streams := relay.ListStreams()
	if len(streams) != 3 {
		t.Errorf("expected 3 streams, got %d", len(streams))
	}
}

func TestListStreamsEmpty(t *testing.T) {
	relay := newTestRelay()

	streams := relay.ListStreams()
	if len(streams) != 0 {
		t.Errorf("expected 0 streams, got %d", len(streams))
	}
}

func TestCloseAll(t *testing.T) {
	relay := newTestRelay()
	ctx := context.Background()

	relay.CreateStream(ctx, "ws-1")
	relay.CreateStream(ctx, "ws-2")
	relay.CreateStream(ctx, "ws-3")

	relay.CloseAll()

	streams := relay.ListStreams()
	if len(streams) != 0 {
		t.Errorf("expected 0 streams after CloseAll, got %d", len(streams))
	}

	// Should be able to create new streams after CloseAll
	_, err := relay.CreateStream(ctx, "ws-1")
	if err != nil {
		t.Fatalf("expected to create stream after CloseAll, got error: %v", err)
	}
}

func TestRouteInput(t *testing.T) {
	relay := newTestRelay()
	ctx := context.Background()

	relay.CreateStream(ctx, "ws-1")

	err := relay.RouteInput(ctx, "ws-1", []byte("mouse_click"))
	if err != nil {
		t.Fatalf("RouteInput returned error: %v", err)
	}
}

func TestRouteInputNoStream(t *testing.T) {
	relay := newTestRelay()
	ctx := context.Background()

	err := relay.RouteInput(ctx, "nonexistent", []byte("mouse_click"))
	if err == nil {
		t.Error("expected error routing input for nonexistent stream, got nil")
	}
}

func TestPauseAndResumeStream(t *testing.T) {
	relay := newTestRelay()
	ctx := context.Background()

	relay.CreateStream(ctx, "ws-1")

	err := relay.PauseStream(ctx, "ws-1")
	if err != nil {
		t.Fatalf("PauseStream returned error: %v", err)
	}

	stream, _ := relay.GetStream("ws-1")
	if stream.Status != StreamPaused {
		t.Errorf("expected status paused, got %s", stream.Status)
	}

	err = relay.ResumeStream(ctx, "ws-1")
	if err != nil {
		t.Fatalf("ResumeStream returned error: %v", err)
	}

	stream, _ = relay.GetStream("ws-1")
	if stream.Status != StreamActive {
		t.Errorf("expected status active, got %s", stream.Status)
	}
}

func TestResumeNonPausedStream(t *testing.T) {
	relay := newTestRelay()
	ctx := context.Background()

	relay.CreateStream(ctx, "ws-1")

	err := relay.ResumeStream(ctx, "ws-1")
	if err == nil {
		t.Error("expected error resuming non-paused stream, got nil")
	}
}