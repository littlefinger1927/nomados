package handler

import (
	"context"

	browserv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/browser_manager/v1"
	"github.com/nomados/nomados/services/browser-manager/internal/chromium"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BrowserManagerServiceGRPCAdapter implements the
// browserv1.BrowserManagerServiceServer gRPC interface by translating between
// proto request/response types and the ChromiumLauncher's instance tracking.
type BrowserManagerServiceGRPCAdapter struct {
	browserv1.UnimplementedBrowserManagerServiceServer
	launcher *chromium.ChromiumLauncher
}

// NewBrowserManagerServiceGRPCAdapter creates a new BrowserManagerServiceGRPCAdapter.
func NewBrowserManagerServiceGRPCAdapter(launcher *chromium.ChromiumLauncher) *BrowserManagerServiceGRPCAdapter {
	return &BrowserManagerServiceGRPCAdapter{
		launcher: launcher,
	}
}

// ListInstances returns all running browser instances.
func (a *BrowserManagerServiceGRPCAdapter) ListInstances(_ context.Context, _ *browserv1.ListInstancesRequest) (*browserv1.ListInstancesResponse, error) {
	instances := a.launcher.ListInstances()

	protoInstances := make([]*browserv1.Instance, 0, len(instances))
	for _, inst := range instances {
		protoInstances = append(protoInstances, &browserv1.Instance{
			WorkspaceId: inst.WorkspaceID,
			Status:      instanceStatusFromBrowser(inst),
			ChromiumPid: int32(inst.PID),
			StartedAt:   inst.StartedAt.Unix(),
		})
	}

	return &browserv1.ListInstancesResponse{
		Instances: protoInstances,
	}, nil
}

// GetInstance returns a single browser instance by workspace ID.
func (a *BrowserManagerServiceGRPCAdapter) GetInstance(_ context.Context, req *browserv1.GetInstanceRequest) (*browserv1.GetInstanceResponse, error) {
	workspaceID := req.GetWorkspaceId().GetValue()
	if workspaceID == "" {
		return nil, status.Error(codes.InvalidArgument, "workspace ID is required")
	}

	inst, exists := a.launcher.GetInstance(workspaceID)
	if !exists {
		return nil, status.Error(codes.NotFound, "browser instance not found")
	}

	return &browserv1.GetInstanceResponse{
		Instance: &browserv1.Instance{
			WorkspaceId: inst.WorkspaceID,
			Status:      instanceStatusFromBrowser(inst),
			ChromiumPid: int32(inst.PID),
			StartedAt:   inst.StartedAt.Unix(),
		},
	}, nil
}

// instanceStatusFromBrowser maps a BrowserInstance to the appropriate
// proto InstanceStatus. Since the launcher only tracks running instances,
// any instance found is considered running.
func instanceStatusFromBrowser(inst *chromium.BrowserInstance) browserv1.InstanceStatus {
	if inst == nil {
		return browserv1.InstanceStatus_INSTANCE_STATUS_UNSPECIFIED
	}
	return browserv1.InstanceStatus_INSTANCE_STATUS_RUNNING
}

// Ensure the adapter implements the BrowserManagerServiceServer interface.
var _ browserv1.BrowserManagerServiceServer = (*BrowserManagerServiceGRPCAdapter)(nil)