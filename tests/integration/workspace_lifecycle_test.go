package integration

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
	workspacev1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/workspace/v1"
)

// TestCreateWorkspace tests creating a workspace and verifying
// the Creating -> Running state transition.
func TestCreateWorkspace(t *testing.T) {
	skipIfUnreachable(t, workspaceServiceAddr, "workspace-orchestrator")

	client, conn := newWorkspaceClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID := "ws-test-user-" + randomSuffix()
	wsName := "test-workspace-" + randomSuffix()

	resp, err := client.Create(ctx, &workspacev1.CreateWorkspaceRequest{
		UserId: &commonv1.UUID{Value: userID},
		Name:   wsName,
	})
	if err != nil {
		t.Fatalf("Create RPC failed: %v", err)
	}

	if resp.Workspace == nil {
		t.Fatal("expected workspace in response")
	}
	if resp.Workspace.GetId().GetValue() == "" {
		t.Error("expected non-empty workspace ID")
	}
	if resp.Workspace.State != workspacev1.WorkspaceState_RUNNING {
		t.Errorf("expected workspace state RUNNING, got %v", resp.Workspace.State)
	}
	if resp.Workspace.Name != wsName {
		t.Errorf("expected workspace name %s, got %s", wsName, resp.Workspace.Name)
	}
	if resp.Workspace.GetUserId().GetValue() != userID {
		t.Errorf("expected user ID %s, got %s", userID, resp.Workspace.GetUserId().GetValue())
	}
}

// TestPauseResumeWorkspace tests the pause/resume lifecycle:
// create -> running, pause -> paused, resume -> running.
func TestPauseResumeWorkspace(t *testing.T) {
	skipIfUnreachable(t, workspaceServiceAddr, "workspace-orchestrator")

	client, conn := newWorkspaceClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a workspace first
	createResp, err := client.Create(ctx, &workspacev1.CreateWorkspaceRequest{
		UserId: &commonv1.UUID{Value: "ws-pause-user-" + randomSuffix()},
		Name:   "pause-test-workspace",
	})
	if err != nil {
		t.Fatalf("Create RPC failed: %v", err)
	}
	wsID := createResp.Workspace.Id

	// Pause the workspace
	pauseResp, err := client.Pause(ctx, &workspacev1.PauseWorkspaceRequest{
		Id: wsID,
	})
	if err != nil {
		t.Fatalf("Pause RPC failed: %v", err)
	}
	if pauseResp.State != workspacev1.WorkspaceState_PAUSED {
		t.Errorf("expected state PAUSED after pause, got %v", pauseResp.State)
	}

	// Resume the workspace
	resumeResp, err := client.Resume(ctx, &workspacev1.ResumeWorkspaceRequest{
		Id: wsID,
	})
	if err != nil {
		t.Fatalf("Resume RPC failed: %v", err)
	}
	if resumeResp.State != workspacev1.WorkspaceState_RUNNING {
		t.Errorf("expected state RUNNING after resume, got %v", resumeResp.State)
	}
}

// TestStopWorkspace tests creating a workspace, stopping it, and verifying
// the Stopped state.
func TestStopWorkspace(t *testing.T) {
	skipIfUnreachable(t, workspaceServiceAddr, "workspace-orchestrator")

	client, conn := newWorkspaceClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a workspace
	createResp, err := client.Create(ctx, &workspacev1.CreateWorkspaceRequest{
		UserId: &commonv1.UUID{Value: "ws-stop-user-" + randomSuffix()},
		Name:   "stop-test-workspace",
	})
	if err != nil {
		t.Fatalf("Create RPC failed: %v", err)
	}
	wsID := createResp.Workspace.Id

	// Stop the workspace
	stopResp, err := client.Stop(ctx, &workspacev1.StopWorkspaceRequest{
		Id: wsID,
	})
	if err != nil {
		t.Fatalf("Stop RPC failed: %v", err)
	}
	if stopResp.State != workspacev1.WorkspaceState_STOPPED {
		t.Errorf("expected state STOPPED after stop, got %v", stopResp.State)
	}
}

// TestDestroyWorkspace tests creating, stopping, and then destroying
// a workspace, verifying cleanup.
func TestDestroyWorkspace(t *testing.T) {
	skipIfUnreachable(t, workspaceServiceAddr, "workspace-orchestrator")

	client, conn := newWorkspaceClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a workspace
	createResp, err := client.Create(ctx, &workspacev1.CreateWorkspaceRequest{
		UserId: &commonv1.UUID{Value: "ws-destroy-user-" + randomSuffix()},
		Name:   "destroy-test-workspace",
	})
	if err != nil {
		t.Fatalf("Create RPC failed: %v", err)
	}
	wsID := createResp.Workspace.Id

	// Stop the workspace first
	_, err = client.Stop(ctx, &workspacev1.StopWorkspaceRequest{
		Id: wsID,
	})
	if err != nil {
		t.Fatalf("Stop RPC failed: %v", err)
	}

	// Destroy the workspace
	_, err = client.Destroy(ctx, &workspacev1.DestroyWorkspaceRequest{
		Id: wsID,
	})
	if err != nil {
		t.Fatalf("Destroy RPC failed: %v", err)
	}

	// Verify the workspace no longer exists
	_, err = client.Get(ctx, &workspacev1.GetWorkspaceRequest{
		Id: wsID,
	})
	if err == nil {
		t.Error("expected error when getting destroyed workspace, got nil")
	}
}

// TestInvalidStateTransition verifies that invalid state transitions
// are rejected. For example, you cannot pause a Stopped workspace.
func TestInvalidStateTransition(t *testing.T) {
	skipIfUnreachable(t, workspaceServiceAddr, "workspace-orchestrator")

	client, conn := newWorkspaceClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create and stop a workspace
	createResp, err := client.Create(ctx, &workspacev1.CreateWorkspaceRequest{
		UserId: &commonv1.UUID{Value: "ws-invalid-user-" + randomSuffix()},
		Name:   "invalid-transition-workspace",
	})
	if err != nil {
		t.Fatalf("Create RPC failed: %v", err)
	}
	wsID := createResp.Workspace.Id

	// Stop the workspace
	_, err = client.Stop(ctx, &workspacev1.StopWorkspaceRequest{
		Id: wsID,
	})
	if err != nil {
		t.Fatalf("Stop RPC failed: %v", err)
	}

	// Attempt to pause a stopped workspace (should fail)
	_, err = client.Pause(ctx, &workspacev1.PauseWorkspaceRequest{
		Id: wsID,
	})
	if err == nil {
		t.Error("expected error when pausing a stopped workspace, got nil")
	}
}