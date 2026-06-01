package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/docker"
)

// TestIsValidTransition tests all valid state transitions.
func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		from     WorkspaceState
		to       WorkspaceState
		expected bool
	}{
		// Valid transitions
		{StateCreating, StateRunning, true},
		{StateRunning, StatePaused, true},
		{StateRunning, StateStopping, true},
		{StatePaused, StateRunning, true},
		{StatePaused, StateStopping, true},
		{StateStopping, StateStopped, true},

		// Invalid transitions
		{StateCreating, StatePaused, false},
		{StateCreating, StateStopped, false},
		{StateCreating, StateStopping, false},
		{StateRunning, StateCreating, false},
		{StateRunning, StateStopped, false},
		{StateRunning, StateRunning, false},
		{StatePaused, StateCreating, false},
		{StatePaused, StatePaused, false},
		{StatePaused, StateStopped, false},
		{StateStopped, StateRunning, false},
		{StateStopped, StatePaused, false},
		{StateStopped, StateCreating, true},
		{StateStopped, StateStopped, false},
		{StateStopping, StateRunning, false},
		{StateStopping, StatePaused, false},
		{StateStopping, StateCreating, false},
	}

	for _, tt := range tests {
		result := IsValidTransition(tt.from, tt.to)
		if result != tt.expected {
			t.Errorf("IsValidTransition(%s, %s) = %v, expected %v", tt.from, tt.to, result, tt.expected)
		}
	}
}

// TestWorkspaceStateConstants verifies state constants.
func TestWorkspaceStateConstants(t *testing.T) {
	states := map[WorkspaceState]string{
		StateCreating: "creating",
		StateRunning:  "running",
		StatePaused:   "paused",
		StateStopping: "stopping",
		StateStopped:  "stopped",
	}
	for state, expected := range states {
		if string(state) != expected {
			t.Errorf("expected %s, got %s", expected, state)
		}
	}
}

func newTestService() (*WorkspaceService, *docker.MockClient) {
	mockDocker := docker.NewMockClient()
	logger := logging.NewLogger("workspace-orchestrator-test", nil)
	svc := NewWorkspaceService(mockDocker, nil, logger, nil)
	return svc, mockDocker
}

func TestCreateWorkspace(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	ws, err := svc.CreateWorkspace(ctx, "user-1", "test-workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.ID == "" {
		t.Error("expected non-empty workspace ID")
	}
	if ws.UserID != "user-1" {
		t.Errorf("expected UserID user-1, got %s", ws.UserID)
	}
	if ws.Name != "test-workspace" {
		t.Errorf("expected Name test-workspace, got %s", ws.Name)
	}
	if ws.State != StateRunning {
		t.Errorf("expected State running, got %s", ws.State)
	}
	if ws.CreatedAt == 0 {
		t.Error("expected non-zero CreatedAt")
	}
	if ws.UpdatedAt == 0 {
		t.Error("expected non-zero UpdatedAt")
	}
}

func TestCreateWorkspaceDockerError(t *testing.T) {
	mockDocker := docker.NewMockClient()
	mockDocker.CreateError = fmt.Errorf("docker unavailable")
	logger := logging.NewLogger("workspace-orchestrator-test", nil)
	svc := NewWorkspaceService(mockDocker, nil, logger, nil)
	ctx := context.Background()

	_, err := svc.CreateWorkspace(ctx, "user-1", "test-workspace")
	if err == nil {
		t.Error("expected error when docker fails, got nil")
	}
}

func TestGetWorkspace(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, err := svc.CreateWorkspace(ctx, "user-1", "test-workspace")
	if err != nil {
		t.Fatalf("unexpected error creating workspace: %v", err)
	}

	ws, err := svc.GetWorkspace(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, ws.ID)
	}
}

func TestGetWorkspaceNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.GetWorkspace(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent workspace, got nil")
	}
}

func TestListWorkspaces(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.CreateWorkspace(ctx, "user-1", "ws-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = svc.CreateWorkspace(ctx, "user-1", "ws-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = svc.CreateWorkspace(ctx, "user-2", "ws-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	workspaces, err := svc.ListWorkspaces(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workspaces) != 2 {
		t.Errorf("expected 2 workspaces for user-1, got %d", len(workspaces))
	}

	workspaces, err = svc.ListWorkspaces(ctx, "user-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workspaces) != 1 {
		t.Errorf("expected 1 workspace for user-2, got %d", len(workspaces))
	}

	workspaces, err = svc.ListWorkspaces(ctx, "user-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workspaces) != 0 {
		t.Errorf("expected 0 workspaces for user-3, got %d", len(workspaces))
	}
}

func TestPauseWorkspace(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, err := svc.CreateWorkspace(ctx, "user-1", "test-workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ws, err := svc.PauseWorkspace(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.State != StatePaused {
		t.Errorf("expected state paused, got %s", ws.State)
	}
}

func TestPauseWorkspaceInvalidState(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateWorkspace(ctx, "user-1", "test-workspace")

	// Pause once (valid)
	_, err := svc.PauseWorkspace(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Pause again (invalid - can't pause a paused workspace)
	_, err = svc.PauseWorkspace(ctx, created.ID)
	if err == nil {
		t.Error("expected error when pausing already paused workspace, got nil")
	}
}

func TestPauseWorkspaceNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.PauseWorkspace(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent workspace, got nil")
	}
}

func TestResumeWorkspace(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateWorkspace(ctx, "user-1", "test-workspace")
	_, err := svc.PauseWorkspace(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ws, err := svc.ResumeWorkspace(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.State != StateRunning {
		t.Errorf("expected state running, got %s", ws.State)
	}
}

func TestResumeWorkspaceInvalidState(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateWorkspace(ctx, "user-1", "test-workspace")

	// Can't resume a running workspace
	_, err := svc.ResumeWorkspace(ctx, created.ID)
	if err == nil {
		t.Error("expected error when resuming running workspace, got nil")
	}
}

func TestStopWorkspace(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateWorkspace(ctx, "user-1", "test-workspace")

	ws, err := svc.StopWorkspace(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.State != StateStopped {
		t.Errorf("expected state stopped, got %s", ws.State)
	}
}

func TestStopPausedWorkspace(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateWorkspace(ctx, "user-1", "test-workspace")
	_, err := svc.PauseWorkspace(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ws, err := svc.StopWorkspace(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.State != StateStopped {
		t.Errorf("expected state stopped, got %s", ws.State)
	}
}

func TestStopWorkspaceInvalidState(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateWorkspace(ctx, "user-1", "test-workspace")

	// Stop once (valid)
	_, err := svc.StopWorkspace(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stop again (invalid - can't stop a stopped workspace)
	_, err = svc.StopWorkspace(ctx, created.ID)
	if err == nil {
		t.Error("expected error when stopping already stopped workspace, got nil")
	}
}

func TestDestroyWorkspace(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateWorkspace(ctx, "user-1", "test-workspace")

	err := svc.DestroyWorkspace(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify workspace is removed
	_, err = svc.GetWorkspace(ctx, created.ID)
	if err == nil {
		t.Error("expected error getting destroyed workspace, got nil")
	}
}

func TestDestroyWorkspaceNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	err := svc.DestroyWorkspace(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent workspace, got nil")
	}
}

func TestFullLifecycle(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	// Create
	ws, err := svc.CreateWorkspace(ctx, "user-1", "lifecycle-test")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ws.State != StateRunning {
		t.Errorf("expected running after create, got %s", ws.State)
	}

	// Pause
	ws, err = svc.PauseWorkspace(ctx, ws.ID)
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	if ws.State != StatePaused {
		t.Errorf("expected paused, got %s", ws.State)
	}

	// Resume
	ws, err = svc.ResumeWorkspace(ctx, ws.ID)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if ws.State != StateRunning {
		t.Errorf("expected running after resume, got %s", ws.State)
	}

	// Stop
	ws, err = svc.StopWorkspace(ctx, ws.ID)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if ws.State != StateStopped {
		t.Errorf("expected stopped, got %s", ws.State)
	}

	// Destroy
	err = svc.DestroyWorkspace(ctx, ws.ID)
	if err != nil {
		t.Fatalf("destroy: %v", err)
	}
}

func TestPauseFromStoppedFails(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateWorkspace(ctx, "user-1", "test-workspace")
	_, _ = svc.StopWorkspace(ctx, created.ID)

	_, err := svc.PauseWorkspace(ctx, created.ID)
	if err == nil {
		t.Error("expected error when pausing stopped workspace, got nil")
	}
}

func TestResumeFromStoppedFails(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateWorkspace(ctx, "user-1", "test-workspace")
	_, _ = svc.StopWorkspace(ctx, created.ID)

	_, err := svc.ResumeWorkspace(ctx, created.ID)
	if err == nil {
		t.Error("expected error when resuming stopped workspace, got nil")
	}
}

func TestStartFromStopped(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	ws, err := svc.CreateWorkspace(ctx, "user-1", "test-ws")
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	// Stop the workspace first
	_, err = svc.StopWorkspace(ctx, ws.ID)
	if err != nil {
		t.Fatalf("StopWorkspace: %v", err)
	}
	// Start it again
	result, err := svc.StartWorkspace(ctx, ws.ID)
	if err != nil {
		t.Fatalf("StartWorkspace: %v", err)
	}
	if result.State != StateRunning {
		t.Errorf("expected Running, got %s", result.State)
	}
}

func TestStartFromRunningFails(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	ws, err := svc.CreateWorkspace(ctx, "user-1", "test-ws")
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	_, err = svc.StartWorkspace(ctx, ws.ID)
	if err == nil {
		t.Error("expected error when starting a running workspace")
	}
}