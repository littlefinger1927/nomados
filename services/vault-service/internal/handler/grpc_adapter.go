package handler

import (
	"context"
	"fmt"

	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
	vaultv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/vault/v1"
	"github.com/nomados/nomados/services/vault-service/internal/keyderivation"
)

// VaultServiceGRPCAdapter wraps VaultServiceHandler and implements the
// vaultv1.VaultServiceServer gRPC interface by translating between
// proto request/response types and the existing handler's struct types.
type VaultServiceGRPCAdapter struct {
	vaultv1.UnimplementedVaultServiceServer
	handler *VaultServiceHandler
}

// NewVaultServiceGRPCAdapter creates a new VaultServiceGRPCAdapter.
func NewVaultServiceGRPCAdapter(handler *VaultServiceHandler) *VaultServiceGRPCAdapter {
	return &VaultServiceGRPCAdapter{
		handler: handler,
	}
}

// DeriveWorkspaceKey handles the DeriveWorkspaceKey RPC.
// The proto takes user_id and workspace_id as UUIDs. The vault service
// derives the key from the master key, but the proto does not carry the
// master key — the client encrypts and sends it out-of-band. For Phase 1,
// we derive from the workspace_id bytes as a placeholder context.
func (a *VaultServiceGRPCAdapter) DeriveWorkspaceKey(ctx context.Context, req *vaultv1.DeriveWorkspaceKeyRequest) (*vaultv1.DeriveWorkspaceKeyResponse, error) {
	workspaceID := req.GetWorkspaceId().GetValue()
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}
	userID := req.GetUserId().GetValue()
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// For Phase 1, derive the workspace key using the user_id as the
	// master key input (this is a placeholder — real implementation will
	// receive the master key from the client via a secure channel).
	masterKey := []byte(userID)

	result, err := a.handler.DeriveWorkspaceKey(ctx, &DeriveWorkspaceKeyRequest{
		MasterKey:   masterKey,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	// Generate a key ID for tracking.
	keyID := keyderivation.DeriveKeyID(workspaceID, "workspace")

	return &vaultv1.DeriveWorkspaceKeyResponse{
		EncryptedKey: result.WorkspaceKey,
		KeyId:        keyID,
	}, nil
}

// DeriveFileKey handles the DeriveFileKey RPC.
func (a *VaultServiceGRPCAdapter) DeriveFileKey(ctx context.Context, req *vaultv1.DeriveFileKeyRequest) (*vaultv1.DeriveFileKeyResponse, error) {
	workspaceID := req.GetWorkspaceId().GetValue()
	fileID := req.GetFileId().GetValue()

	if workspaceID == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}
	if fileID == "" {
		return nil, fmt.Errorf("file ID is required")
	}

	// For Phase 1, derive the workspace key first from workspace_id as context,
	// then derive the file key. The workspace key acts as the parent key.
	workspaceKeyCtx := keyderivation.DeriveWorkspaceKeyContext(workspaceID)

	result, err := a.handler.DeriveFileKey(ctx, &DeriveFileKeyRequest{
		WorkspaceKey: workspaceKeyCtx,
		FileID:       fileID,
	})
	if err != nil {
		return nil, err
	}

	// Generate a key ID for tracking.
	keyID := keyderivation.DeriveKeyID(fileID, "file")

	return &vaultv1.DeriveFileKeyResponse{
		EncryptedKey: result.FileKey,
		KeyId:        keyID,
	}, nil
}

// RotateWorkspaceKey handles the RotateWorkspaceKey RPC.
func (a *VaultServiceGRPCAdapter) RotateWorkspaceKey(ctx context.Context, req *vaultv1.RotateWorkspaceKeyRequest) (*commonv1.Empty, error) {
	workspaceID := req.GetWorkspaceId().GetValue()
	userID := req.GetUserId().GetValue()

	if workspaceID == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// For Phase 1, derive a new master key from user_id as placeholder.
	masterKey := []byte(userID)

	_, err := a.handler.RotateWorkspaceKey(ctx, &RotateWorkspaceKeyRequest{
		MasterKey:   masterKey,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	return &commonv1.Empty{}, nil
}

// Ensure the adapter implements the VaultServiceServer interface.
var _ vaultv1.VaultServiceServer = (*VaultServiceGRPCAdapter)(nil)