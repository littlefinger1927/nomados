package docker

import (
	"context"
	"fmt"
	"testing"
)

func TestNewMockClient(t *testing.T) {
	mock := NewMockClient()
	if mock == nil {
		t.Fatal("expected non-nil MockClient")
	}
	if len(mock.Workspaces) != 0 {
		t.Errorf("expected empty Workspaces map, got %d entries", len(mock.Workspaces))
	}
}

func TestMockClientCreateWorkspace(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	err := mock.CreateWorkspace(ctx, "ws-123", "user-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.Workspaces["ws-123"] != string(StateRunning) {
		t.Errorf("expected workspace state %s, got %s", StateRunning, mock.Workspaces["ws-123"])
	}
}

func TestMockClientCreateWorkspaceError(t *testing.T) {
	mock := NewMockClient()
	mock.CreateError = fmt.Errorf("docker unavailable")
	ctx := context.Background()

	err := mock.CreateWorkspace(ctx, "ws-123", "user-456")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestMockClientPauseWorkspace(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	_ = mock.CreateWorkspace(ctx, "ws-123", "user-456")

	err := mock.PauseWorkspace(ctx, "ws-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.Workspaces["ws-123"] != string(StatePaused) {
		t.Errorf("expected state %s, got %s", StatePaused, mock.Workspaces["ws-123"])
	}
}

func TestMockClientPauseWorkspaceNotFound(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	err := mock.PauseWorkspace(ctx, "ws-nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent workspace, got nil")
	}
}

func TestMockClientResumeWorkspace(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	_ = mock.CreateWorkspace(ctx, "ws-123", "user-456")
	_ = mock.PauseWorkspace(ctx, "ws-123")

	err := mock.ResumeWorkspace(ctx, "ws-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.Workspaces["ws-123"] != string(StateRunning) {
		t.Errorf("expected state %s, got %s", StateRunning, mock.Workspaces["ws-123"])
	}
}

func TestMockClientStopWorkspace(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	_ = mock.CreateWorkspace(ctx, "ws-123", "user-456")

	err := mock.StopWorkspace(ctx, "ws-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.Workspaces["ws-123"] != string(StateStopped) {
		t.Errorf("expected state %s, got %s", StateStopped, mock.Workspaces["ws-123"])
	}
}

func TestMockClientRemoveWorkspace(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	_ = mock.CreateWorkspace(ctx, "ws-123", "user-456")

	err := mock.RemoveWorkspace(ctx, "ws-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := mock.Workspaces["ws-123"]; ok {
		t.Error("expected workspace to be removed")
	}
}

func TestMockClientGetWorkspaceStatus(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	_ = mock.CreateWorkspace(ctx, "ws-123", "user-456")

	state, err := mock.GetWorkspaceStatus(ctx, "ws-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != StateRunning {
		t.Errorf("expected state %s, got %s", StateRunning, state)
	}
}

func TestMockClientGetWorkspaceStatusNotFound(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	_, err := mock.GetWorkspaceStatus(ctx, "ws-nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent workspace, got nil")
	}
}

func TestContainerStateConstants(t *testing.T) {
	states := map[ContainerState]string{
		StateCreating: "creating",
		StateRunning:  "running",
		StatePaused:   "paused",
		StateStopped:  "stopped",
		StateUnknown:  "unknown",
	}
	for state, expected := range states {
		if string(state) != expected {
			t.Errorf("expected %s, got %s", expected, state)
		}
	}
}