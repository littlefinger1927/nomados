package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/vault-service/internal/keyderivation"
)

// Request and response types for the vault handler.
// The gRPC adapter translates between proto types and these internal types.

// DeriveWorkspaceKeyRequest represents a request to derive a workspace key.
type DeriveWorkspaceKeyRequest struct {
	MasterKey   []byte
	WorkspaceID string
}

// DeriveWorkspaceKeyResponse represents the response with the derived workspace key.
type DeriveWorkspaceKeyResponse struct {
	WorkspaceKey []byte
}

// DeriveFileKeyRequest represents a request to derive a file key.
type DeriveFileKeyRequest struct {
	WorkspaceKey []byte
	FileID       string
}

// DeriveFileKeyResponse represents the response with the derived file key.
type DeriveFileKeyResponse struct {
	FileKey []byte
}

// RotateWorkspaceKeyRequest represents a request to rotate a workspace key.
type RotateWorkspaceKeyRequest struct {
	MasterKey   []byte
	WorkspaceID string
}

// RotateWorkspaceKeyResponse represents the response with the new workspace key.
type RotateWorkspaceKeyResponse struct {
	NewKey    []byte
	KeyID     string
	RotatedAt time.Time
}

// KeyRotationPublisher defines the interface for publishing key rotation events.
type KeyRotationPublisher interface {
	PublishKeyRotated(ctx context.Context, workspaceID string, keyID string) error
}

// VaultServiceHandler handles vault key derivation operations.
type VaultServiceHandler struct {
	kd        *keyderivation.KeyDeriver
	publisher KeyRotationPublisher
	logger    *logging.Logger
}

// NewVaultServiceHandler creates a new VaultServiceHandler.
func NewVaultServiceHandler(kd *keyderivation.KeyDeriver, publisher KeyRotationPublisher, logger *logging.Logger) *VaultServiceHandler {
	return &VaultServiceHandler{
		kd:        kd,
		publisher: publisher,
		logger:    logger.With("component", "vault-handler"),
	}
}

// DeriveWorkspaceKey derives a workspace encryption key from a master key and workspace ID.
// The master key buffer is zeroized after the operation for security.
func (h *VaultServiceHandler) DeriveWorkspaceKey(ctx context.Context, req *DeriveWorkspaceKeyRequest) (*DeriveWorkspaceKeyResponse, error) {
	if len(req.MasterKey) == 0 {
		return nil, fmt.Errorf("master key is required")
	}
	if req.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	// Session validation is performed at the gRPC adapter layer before
	// reaching this handler.

	h.logger.Info("deriving workspace key", "workspace_id", req.WorkspaceID)

	workspaceKey, err := h.kd.DeriveWorkspaceKey(req.MasterKey, []byte(req.WorkspaceID))
	if err != nil {
		return nil, fmt.Errorf("failed to derive workspace key: %w", err)
	}

	// Zeroize master key after derivation to minimize exposure.
	keyderivation.Zeroize(req.MasterKey)

	h.logger.Info("workspace key derived successfully", "workspace_id", req.WorkspaceID)

	return &DeriveWorkspaceKeyResponse{
		WorkspaceKey: workspaceKey,
	}, nil
}

// DeriveFileKey derives a file encryption key from a workspace key and file ID.
// The workspace key buffer is zeroized after the operation for security.
func (h *VaultServiceHandler) DeriveFileKey(ctx context.Context, req *DeriveFileKeyRequest) (*DeriveFileKeyResponse, error) {
	if len(req.WorkspaceKey) == 0 {
		return nil, fmt.Errorf("workspace key is required")
	}
	if req.FileID == "" {
		return nil, fmt.Errorf("file ID is required")
	}

	// Session validation is performed at the gRPC adapter layer.

	h.logger.Info("deriving file key", "file_id", req.FileID)

	fileKey, err := h.kd.DeriveFileKey(req.WorkspaceKey, []byte(req.FileID))
	if err != nil {
		return nil, fmt.Errorf("failed to derive file key: %w", err)
	}

	// Zeroize workspace key after derivation to minimize exposure.
	keyderivation.Zeroize(req.WorkspaceKey)

	h.logger.Info("file key derived successfully", "file_id", req.FileID)

	return &DeriveFileKeyResponse{
		FileKey: fileKey,
	}, nil
}

// RotateWorkspaceKey rotates a workspace key by deriving a new key and publishing
// a rotation event. The old key should remain valid until all file keys have been
// re-encrypted with the new workspace key.
func (h *VaultServiceHandler) RotateWorkspaceKey(ctx context.Context, req *RotateWorkspaceKeyRequest) (*RotateWorkspaceKeyResponse, error) {
	if len(req.MasterKey) == 0 {
		return nil, fmt.Errorf("master key is required")
	}
	if req.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}

	// Session validation is performed at the gRPC adapter layer.

	h.logger.Info("rotating workspace key", "workspace_id", req.WorkspaceID)

	newKey, err := h.kd.RotateWorkspaceKey(req.MasterKey, []byte(req.WorkspaceID))
	if err != nil {
		return nil, fmt.Errorf("failed to rotate workspace key: %w", err)
	}

	// Zeroize master key after rotation.
	keyderivation.Zeroize(req.MasterKey)

	// Generate a key ID for tracking.
	keyID := uuid.New().String()

	// Publish rotation event to NATS.
	if h.publisher != nil {
		if err := h.publisher.PublishKeyRotated(ctx, req.WorkspaceID, keyID); err != nil {
			h.logger.Error("failed to publish key rotation event", "error", err)
			// Don't fail the rotation if publishing fails; the key is still valid.
		}
	}

	h.logger.Info("workspace key rotated successfully", "workspace_id", req.WorkspaceID, "key_id", keyID)

	return &RotateWorkspaceKeyResponse{
		NewKey:    newKey,
		KeyID:     keyID,
		RotatedAt: time.Now().UTC(),
	}, nil
}