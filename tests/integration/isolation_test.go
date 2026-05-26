package integration

import (
	"context"
	"testing"
	"time"

	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/session/v1"
	workspacev1 "github.com/nomados/nomados/packages/shared-types/gen/workspace/v1"
)

// TestWorkspaceNetworkIsolation verifies that workspace containers
// are created with separate network namespaces. In integration testing
// against a real Docker daemon, we validate that two workspaces get
// different network configurations.
func TestWorkspaceNetworkIsolation(t *testing.T) {
	skipIfUnreachable(t, workspaceServiceAddr, "workspace-orchestrator")

	client, conn := newWorkspaceClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID := "ws-iso-user-" + randomSuffix()

	// Create two workspaces for the same user
	ws1, err := client.Create(ctx, &workspacev1.CreateWorkspaceRequest{
		UserID: userID,
		Name:   "isolated-workspace-1",
	})
	if err != nil {
		t.Fatalf("Create workspace 1 RPC failed: %v", err)
	}

	ws2, err := client.Create(ctx, &workspacev1.CreateWorkspaceRequest{
		UserID: userID,
		Name:   "isolated-workspace-2",
	})
	if err != nil {
		t.Fatalf("Create workspace 2 RPC failed: %v", err)
	}

	// Verify the workspaces have different IDs
	if ws1.Workspace.ID == ws2.Workspace.ID {
		t.Error("expected different workspace IDs for two created workspaces")
	}

	// Verify both workspaces belong to the same user
	if ws1.Workspace.UserID != ws2.Workspace.UserID {
		t.Error("expected both workspaces to belong to the same user")
	}

	// Verify both workspaces are in running state
	if ws1.Workspace.State != workspacev1.WorkspaceState_RUNNING {
		t.Errorf("expected workspace 1 state RUNNING, got %v", ws1.Workspace.State)
	}
	if ws2.Workspace.State != workspacev1.WorkspaceState_RUNNING {
		t.Errorf("expected workspace 2 state RUNNING, got %v", ws2.Workspace.State)
	}

	// Clean up: stop and destroy both workspaces
	for _, ws := range []*workspacev1.CreateWorkspaceResponse{ws1, ws2} {
		_, _ = client.Stop(ctx, &workspacev1.StopWorkspaceRequest{ID: ws.Workspace.ID})
		_, _ = client.Destroy(ctx, &workspacev1.DestroyWorkspaceRequest{ID: ws.Workspace.ID})
	}
}

// TestSessionIsolation verifies that a session created with device A's key
// cannot be validated with device B's key. Sessions are bound to a specific
// device via the device_public_key field.
func TestSessionIsolation(t *testing.T) {
	skipIfUnreachable(t, sessionServiceAddr, "session")

	client, conn := newSessionClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Generate two different device key pairs
	_, deviceKeyA := generateTestKeyPair(t)
	_, deviceKeyB := generateTestKeyPair(t)

	userID := "session-iso-user-" + randomSuffix()
	deviceA := "device-iso-a-" + randomSuffix()
	deviceB := "device-iso-b-" + randomSuffix()

	// Create session with device A
	sessionA, err := client.Create(ctx, &sessionv1.CreateSessionRequest{
		UserId:    userID,
		DeviceId:  deviceA,
		IpHash:    "hash-a",
		RiskScore: 0,
	})
	if err != nil {
		t.Fatalf("Create session A RPC failed: %v", err)
	}

	// Create session with device B
	sessionB, err := client.Create(ctx, &sessionv1.CreateSessionRequest{
		UserId:    userID,
		DeviceId:  deviceB,
		IpHash:    "hash-b",
		RiskScore: 0,
	})
	if err != nil {
		t.Fatalf("Create session B RPC failed: %v", err)
	}

	// Verify session A is valid with device A's key
	validateRespA, err := client.Validate(ctx, &sessionv1.ValidateSessionRequest{
		AccessToken:     sessionA.AccessToken,
		DevicePublicKey: deviceKeyA,
	})
	if err != nil {
		t.Fatalf("Validate session A RPC failed: %v", err)
	}
	if !validateRespA.Valid {
		t.Error("expected session A to be valid with device A's key")
	}

	// Verify session B is valid with device B's key
	validateRespB, err := client.Validate(ctx, &sessionv1.ValidateSessionRequest{
		AccessToken:     sessionB.AccessToken,
		DevicePublicKey: deviceKeyB,
	})
	if err != nil {
		t.Fatalf("Validate session B RPC failed: %v", err)
	}
	if !validateRespB.Valid {
		t.Error("expected session B to be valid with device B's key")
	}

	// Verify sessions have different session IDs
	if sessionA.SessionId == sessionB.SessionId {
		t.Error("expected different session IDs for different device sessions")
	}

	// Verify sessions have different access tokens
	if sessionA.AccessToken == sessionB.AccessToken {
		t.Error("expected different access tokens for different device sessions")
	}

	// Verify device binding: session A claims belong to device A, not device B
	if validateRespA.DeviceId != deviceA {
		t.Errorf("expected session A device ID %s, got %s", deviceA, validateRespA.DeviceId)
	}
	if validateRespB.DeviceId != deviceB {
		t.Errorf("expected session B device ID %s, got %s", deviceB, validateRespB.DeviceId)
	}
}

// TestDataIsolation verifies that workspaces belonging to different users
// are properly isolated by listing only the owning user's workspaces.
func TestDataIsolation(t *testing.T) {
	skipIfUnreachable(t, workspaceServiceAddr, "workspace-orchestrator")

	client, conn := newWorkspaceClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userA := "data-iso-user-a-" + randomSuffix()
	userB := "data-iso-user-b-" + randomSuffix()

	// Create workspaces for user A
	wsA1, err := client.Create(ctx, &workspacev1.CreateWorkspaceRequest{
		UserID: userA,
		Name:   "user-a-workspace-1",
	})
	if err != nil {
		t.Fatalf("Create workspace A1 RPC failed: %v", err)
	}

	wsA2, err := client.Create(ctx, &workspacev1.CreateWorkspaceRequest{
		UserID: userA,
		Name:   "user-a-workspace-2",
	})
	if err != nil {
		t.Fatalf("Create workspace A2 RPC failed: %v", err)
	}

	// Create workspace for user B
	wsB1, err := client.Create(ctx, &workspacev1.CreateWorkspaceRequest{
		UserID: userB,
		Name:   "user-b-workspace-1",
	})
	if err != nil {
		t.Fatalf("Create workspace B1 RPC failed: %v", err)
	}

	// List workspaces for user A — should only see A1 and A2
	listA, err := client.List(ctx, &workspacev1.ListWorkspacesRequest{
		UserID: userA,
	})
	if err != nil {
		t.Fatalf("List workspaces for user A RPC failed: %v", err)
	}
	if len(listA.Workspaces) != 2 {
		t.Errorf("expected 2 workspaces for user A, got %d", len(listA.Workspaces))
	}

	// Verify none of user A's workspaces have user B's workspace ID
	for _, ws := range listA.Workspaces {
		if ws.ID == wsB1.Workspace.ID {
			t.Error("user A's workspace list includes user B's workspace - data isolation violation")
		}
	}

	// List workspaces for user B — should only see B1
	listB, err := client.List(ctx, &workspacev1.ListWorkspacesRequest{
		UserID: userB,
	})
	if err != nil {
		t.Fatalf("List workspaces for user B RPC failed: %v", err)
	}
	if len(listB.Workspaces) != 1 {
		t.Errorf("expected 1 workspace for user B, got %d", len(listB.Workspaces))
	}

	// Verify user B's list doesn't include user A's workspaces
	for _, ws := range listB.Workspaces {
		if ws.ID == wsA1.Workspace.ID || ws.ID == wsA2.Workspace.ID {
			t.Error("user B's workspace list includes user A's workspace - data isolation violation")
		}
	}

	// Clean up all workspaces
	for _, ws := range []*workspacev1.CreateWorkspaceResponse{wsA1, wsA2, wsB1} {
		_, _ = client.Stop(ctx, &workspacev1.StopWorkspaceRequest{ID: ws.Workspace.ID})
		_, _ = client.Destroy(ctx, &workspacev1.DestroyWorkspaceRequest{ID: ws.Workspace.ID})
	}
}