package handler

import (
	"context"

	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/common/v1"
	workspacev1 "github.com/nomados/nomados/packages/shared-types/gen/workspace/v1"
	"github.com/nomados/nomados/services/workspace-orchestrator/internal/service"
)

// WorkspaceServiceHandler implements the WorkspaceServiceServer gRPC interface.
type WorkspaceServiceHandler struct {
	workspacev1.UnimplementedWorkspaceServiceServer
	svc *service.WorkspaceService
}

// NewWorkspaceServiceHandler creates a new WorkspaceServiceHandler.
func NewWorkspaceServiceHandler(svc *service.WorkspaceService) *WorkspaceServiceHandler {
	return &WorkspaceServiceHandler{
		svc: svc,
	}
}

// toProtoWorkspace converts an internal Workspace to a proto Workspace.
func toProtoWorkspace(ws *service.Workspace) *workspacev1.Workspace {
	var state workspacev1.WorkspaceState
	switch ws.State {
	case service.StateCreating:
		state = workspacev1.WorkspaceState_CREATING
	case service.StateRunning:
		state = workspacev1.WorkspaceState_RUNNING
	case service.StatePaused:
		state = workspacev1.WorkspaceState_PAUSED
	case service.StateStopping:
		state = workspacev1.WorkspaceState_STOPPING
	case service.StateStopped:
		state = workspacev1.WorkspaceState_STOPPED
	default:
		state = workspacev1.WorkspaceState_UNSPECIFIED
	}

	return &workspacev1.Workspace{
		ID:        ws.ID,
		UserID:    ws.UserID,
		Name:      ws.Name,
		State:     state,
		CreatedAt: ws.CreatedAt,
		UpdatedAt: ws.UpdatedAt,
	}
}

// Create handles the gRPC Create RPC.
func (h *WorkspaceServiceHandler) Create(ctx context.Context, req *workspacev1.CreateWorkspaceRequest) (*workspacev1.CreateWorkspaceResponse, error) {
	ws, err := h.svc.CreateWorkspace(ctx, req.UserID, req.Name)
	if err != nil {
		return nil, err
	}

	return &workspacev1.CreateWorkspaceResponse{
		Workspace: toProtoWorkspace(ws),
	}, nil
}

// Get handles the gRPC Get RPC.
func (h *WorkspaceServiceHandler) Get(ctx context.Context, req *workspacev1.GetWorkspaceRequest) (*workspacev1.Workspace, error) {
	ws, err := h.svc.GetWorkspace(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	return toProtoWorkspace(ws), nil
}

// List handles the gRPC List RPC.
func (h *WorkspaceServiceHandler) List(ctx context.Context, req *workspacev1.ListWorkspacesRequest) (*workspacev1.ListWorkspacesResponse, error) {
	workspaces, err := h.svc.ListWorkspaces(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	protoWorkspaces := make([]*workspacev1.Workspace, len(workspaces))
	for i, ws := range workspaces {
		protoWorkspaces[i] = toProtoWorkspace(ws)
	}

	return &workspacev1.ListWorkspacesResponse{
		Workspaces: protoWorkspaces,
	}, nil
}

// Pause handles the gRPC Pause RPC.
func (h *WorkspaceServiceHandler) Pause(ctx context.Context, req *workspacev1.PauseWorkspaceRequest) (*workspacev1.Workspace, error) {
	ws, err := h.svc.PauseWorkspace(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	return toProtoWorkspace(ws), nil
}

// Resume handles the gRPC Resume RPC.
func (h *WorkspaceServiceHandler) Resume(ctx context.Context, req *workspacev1.ResumeWorkspaceRequest) (*workspacev1.Workspace, error) {
	ws, err := h.svc.ResumeWorkspace(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	return toProtoWorkspace(ws), nil
}

// Stop handles the gRPC Stop RPC.
func (h *WorkspaceServiceHandler) Stop(ctx context.Context, req *workspacev1.StopWorkspaceRequest) (*workspacev1.Workspace, error) {
	ws, err := h.svc.StopWorkspace(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	return toProtoWorkspace(ws), nil
}

// Destroy handles the gRPC Destroy RPC.
func (h *WorkspaceServiceHandler) Destroy(ctx context.Context, req *workspacev1.DestroyWorkspaceRequest) (*commonv1.Empty, error) {
	if err := h.svc.DestroyWorkspace(ctx, req.ID); err != nil {
		return nil, err
	}

	return &commonv1.Empty{}, nil
}