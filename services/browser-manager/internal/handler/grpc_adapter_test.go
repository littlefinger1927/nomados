package handler

import (
	"context"
	"testing"

	browserv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/browser_manager/v1"
	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
	"github.com/nomados/nomados/services/browser-manager/internal/chromium"
	"time"
)

func TestListInstances_Empty(t *testing.T) {
	launcher := chromium.NewChromiumLauncher(chromium.LauncherConfig{}, nil)
	adapter := NewBrowserManagerServiceGRPCAdapter(launcher)

	resp, err := adapter.ListInstances(context.Background(), &browserv1.ListInstancesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Instances) != 0 {
		t.Errorf("expected 0 instances, got %d", len(resp.Instances))
	}
}

func TestGetInstance_NotFound(t *testing.T) {
	launcher := chromium.NewChromiumLauncher(chromium.LauncherConfig{}, nil)
	adapter := NewBrowserManagerServiceGRPCAdapter(launcher)

	_, err := adapter.GetInstance(context.Background(), &browserv1.GetInstanceRequest{
		WorkspaceId: &commonv1.UUID{Value: "ws-nonexistent"},
	})

	if err == nil {
		t.Fatal("expected error for nonexistent workspace, got nil")
	}
}

func TestGetInstance_EmptyWorkspaceID(t *testing.T) {
	launcher := chromium.NewChromiumLauncher(chromium.LauncherConfig{}, nil)
	adapter := NewBrowserManagerServiceGRPCAdapter(launcher)

	_, err := adapter.GetInstance(context.Background(), &browserv1.GetInstanceRequest{
		WorkspaceId: &commonv1.UUID{Value: ""},
	})

	if err == nil {
		t.Fatal("expected error for empty workspace ID, got nil")
	}
}

func TestListInstances_WithRunningInstance(t *testing.T) {
	launcher := chromium.NewChromiumLauncher(chromium.LauncherConfig{}, nil)
	adapter := NewBrowserManagerServiceGRPCAdapter(launcher)

	// Manually store an instance to simulate a running browser
	now := time.Now()
	launcher.StoreInstanceForTest("ws-test-123", &chromium.BrowserInstance{
		PID:         99999,
		WorkspaceID: "ws-test-123",
		Fingerprint: chromium.GenerateFingerprint("ws-test-123"),
		ProfilePath:  "/tmp/test-profiles/ws-test-123",
		StartedAt:   now,
	})

	resp, err := adapter.ListInstances(context.Background(), &browserv1.ListInstancesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(resp.Instances))
	}

	inst := resp.Instances[0]
	if inst.WorkspaceId != "ws-test-123" {
		t.Errorf("expected workspace ID ws-test-123, got %s", inst.WorkspaceId)
	}
	if inst.ChromiumPid != 99999 {
		t.Errorf("expected PID 99999, got %d", inst.ChromiumPid)
	}
	if inst.Status != browserv1.InstanceStatus_INSTANCE_STATUS_RUNNING {
		t.Errorf("expected status RUNNING, got %v", inst.Status)
	}
	if inst.StartedAt == 0 {
		t.Error("expected non-zero started_at timestamp")
	}
}

func TestGetInstance_Found(t *testing.T) {
	launcher := chromium.NewChromiumLauncher(chromium.LauncherConfig{}, nil)
	adapter := NewBrowserManagerServiceGRPCAdapter(launcher)

	now := time.Now()
	launcher.StoreInstanceForTest("ws-test-456", &chromium.BrowserInstance{
		PID:         77777,
		WorkspaceID: "ws-test-456",
		Fingerprint: chromium.GenerateFingerprint("ws-test-456"),
		ProfilePath:  "/tmp/test-profiles/ws-test-456",
		StartedAt:   now,
	})

	resp, err := adapter.GetInstance(context.Background(), &browserv1.GetInstanceRequest{
		WorkspaceId: &commonv1.UUID{Value: "ws-test-456"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Instance.WorkspaceId != "ws-test-456" {
		t.Errorf("expected workspace ID ws-test-456, got %s", resp.Instance.WorkspaceId)
	}
	if resp.Instance.ChromiumPid != 77777 {
		t.Errorf("expected PID 77777, got %d", resp.Instance.ChromiumPid)
	}
}