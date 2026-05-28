package handler

import (
	"context"
	"fmt"
	"strings"

	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
	vaultv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/vault/v1"
	"github.com/nomados/nomados/services/vault-service/internal/keyderivation"
	"github.com/nomados/nomados/services/vault-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// VaultServiceGRPCAdapter wraps VaultServiceHandler and implements the
// vaultv1.VaultServiceServer gRPC interface by translating between
// proto request/response types and the existing handler's struct types.
type VaultServiceGRPCAdapter struct {
	vaultv1.UnimplementedVaultServiceServer
	handler          *VaultServiceHandler
	sessionValidator *service.SessionValidator
}

// NewVaultServiceGRPCAdapter creates a new VaultServiceGRPCAdapter.
func NewVaultServiceGRPCAdapter(handler *VaultServiceHandler, sessionValidator *service.SessionValidator) *VaultServiceGRPCAdapter {
	return &VaultServiceGRPCAdapter{
		handler:          handler,
		sessionValidator: sessionValidator,
	}
}

// extractSessionToken extracts the session token from gRPC metadata.
// It looks for an "authorization" metadata entry with a "Bearer" prefix.
func extractSessionToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("no metadata in context")
	}
	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return "", fmt.Errorf("no authorization header")
	}
	token := strings.TrimPrefix(tokens[0], "Bearer ")
	if token == tokens[0] {
		// No "Bearer " prefix was present; use the raw value.
		return tokens[0], nil
	}
	return token, nil
}

// DeriveWorkspaceKey handles the DeriveWorkspaceKey RPC.
// It validates the session token from gRPC metadata, then uses the
// client-provided master key to derive a workspace key.
func (a *VaultServiceGRPCAdapter) DeriveWorkspaceKey(ctx context.Context, req *vaultv1.DeriveWorkspaceKeyRequest) (*vaultv1.DeriveWorkspaceKeyResponse, error) {
	// Validate session token from gRPC metadata.
	sessionToken, err := extractSessionToken(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "session token required")
	}
	_, err = a.sessionValidator.Validate(ctx, sessionToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid session: "+err.Error())
	}

	workspaceID := req.GetWorkspaceId().GetValue()
	if workspaceID == "" {
		return nil, status.Error(codes.InvalidArgument, "workspace ID is required")
	}

	// Use client-provided master key instead of placeholder.
	masterKey := req.GetMasterKey()
	if len(masterKey) == 0 {
		return nil, status.Error(codes.InvalidArgument, "master key is required")
	}

	result, err := a.handler.DeriveWorkspaceKey(ctx, &DeriveWorkspaceKeyRequest{
		MasterKey:   masterKey,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to derive workspace key: %v", err)
	}

	// Generate a key ID for tracking.
	keyID := keyderivation.DeriveKeyID(workspaceID, "workspace")

	return &vaultv1.DeriveWorkspaceKeyResponse{
		EncryptedKey: result.WorkspaceKey,
		KeyId:        keyID,
	}, nil
}

// DeriveFileKey handles the DeriveFileKey RPC.
// It validates the session token, then derives a file key using the
// client-provided master key to first derive the workspace key.
func (a *VaultServiceGRPCAdapter) DeriveFileKey(ctx context.Context, req *vaultv1.DeriveFileKeyRequest) (*vaultv1.DeriveFileKeyResponse, error) {
	// Validate session token from gRPC metadata.
	sessionToken, err := extractSessionToken(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "session token required")
	}
	_, err = a.sessionValidator.Validate(ctx, sessionToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid session: "+err.Error())
	}

	workspaceID := req.GetWorkspaceId().GetValue()
	fileID := req.GetFileId().GetValue()

	if workspaceID == "" {
		return nil, status.Error(codes.InvalidArgument, "workspace ID is required")
	}
	if fileID == "" {
		return nil, status.Error(codes.InvalidArgument, "file ID is required")
	}

	// Use client-provided master key instead of deriving from workspace_id context.
	masterKey := req.GetMasterKey()
	if len(masterKey) == 0 {
		return nil, status.Error(codes.InvalidArgument, "master key is required")
	}

	// Derive workspace key from master key, then derive file key from it.
	workspaceKeyResult, err := a.handler.DeriveWorkspaceKey(ctx, &DeriveWorkspaceKeyRequest{
		MasterKey:   masterKey,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to derive workspace key for file key: %v", err)
	}

	result, err := a.handler.DeriveFileKey(ctx, &DeriveFileKeyRequest{
		WorkspaceKey: workspaceKeyResult.WorkspaceKey,
		FileID:       fileID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to derive file key: %v", err)
	}

	// Generate a key ID for tracking.
	keyID := keyderivation.DeriveKeyID(fileID, "file")

	return &vaultv1.DeriveFileKeyResponse{
		EncryptedKey: result.FileKey,
		KeyId:        keyID,
	}, nil
}

// RotateWorkspaceKey handles the RotateWorkspaceKey RPC.
// It validates the session token, then rotates the workspace key using the
// client-provided master key.
func (a *VaultServiceGRPCAdapter) RotateWorkspaceKey(ctx context.Context, req *vaultv1.RotateWorkspaceKeyRequest) (*commonv1.Empty, error) {
	// Validate session token from gRPC metadata.
	sessionToken, err := extractSessionToken(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "session token required")
	}
	_, err = a.sessionValidator.Validate(ctx, sessionToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid session: "+err.Error())
	}

	workspaceID := req.GetWorkspaceId().GetValue()
	if workspaceID == "" {
		return nil, status.Error(codes.InvalidArgument, "workspace ID is required")
	}

	// Use client-provided master key instead of placeholder.
	masterKey := req.GetMasterKey()
	if len(masterKey) == 0 {
		return nil, status.Error(codes.InvalidArgument, "master key is required")
	}

	_, err = a.handler.RotateWorkspaceKey(ctx, &RotateWorkspaceKeyRequest{
		MasterKey:   masterKey,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to rotate workspace key: %v", err)
	}

	return &commonv1.Empty{}, nil
}

// Ensure the adapter implements the VaultServiceServer interface.
var _ vaultv1.VaultServiceServer = (*VaultServiceGRPCAdapter)(nil)