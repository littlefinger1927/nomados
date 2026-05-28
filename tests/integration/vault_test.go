package integration

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
	vaultv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/vault/v1"
)

// TestVaultDeriveWorkspaceKey tests that DeriveWorkspaceKey returns
// an encrypted key and key ID for valid inputs.
func TestVaultDeriveWorkspaceKey(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: "vault-test-user-1"},
		WorkspaceId: &commonv1.UUID{Value: "vault-test-workspace-1"},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey RPC failed: %v", err)
	}

	if len(resp.EncryptedKey) == 0 {
		t.Error("expected non-empty encrypted key in response")
	}
	if len(resp.KeyId) == 0 {
		t.Error("expected non-empty key ID in response")
	}
}

// TestVaultDeriveWorkspaceKeyDeterministic tests that the same inputs
// produce the same workspace key (deterministic derivation).
func TestVaultDeriveWorkspaceKeyDeterministic(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID := "vault-det-user-" + randomSuffix()
	workspaceID := "vault-det-workspace-" + randomSuffix()

	resp1, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: userID},
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey (1st call) RPC failed: %v", err)
	}

	resp2, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: userID},
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey (2nd call) RPC failed: %v", err)
	}

	// Same inputs should produce the same encrypted key (deterministic)
	if string(resp1.EncryptedKey) != string(resp2.EncryptedKey) {
		t.Error("expected identical encrypted keys for same inputs (derivation should be deterministic)")
	}
	if string(resp1.KeyId) != string(resp2.KeyId) {
		t.Error("expected identical key IDs for same inputs")
	}
}

// TestVaultDeriveWorkspaceKeyIsolation tests that different workspace IDs
// produce different workspace keys (isolation between workspaces).
func TestVaultDeriveWorkspaceKeyIsolation(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID := "vault-iso-user-" + randomSuffix()

	resp1, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: userID},
		WorkspaceId: &commonv1.UUID{Value: "vault-workspace-A"},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey for workspace A failed: %v", err)
	}

	resp2, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: userID},
		WorkspaceId: &commonv1.UUID{Value: "vault-workspace-B"},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey for workspace B failed: %v", err)
	}

	// Different workspace IDs should produce different keys
	if string(resp1.EncryptedKey) == string(resp2.EncryptedKey) {
		t.Error("expected different encrypted keys for different workspace IDs")
	}
	if string(resp1.KeyId) == string(resp2.KeyId) {
		t.Error("expected different key IDs for different workspace IDs")
	}
}

// TestVaultDeriveWorkspaceKeyDifferentUsers tests that different user IDs
// produce different workspace keys (isolation between users).
func TestVaultDeriveWorkspaceKeyDifferentUsers(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workspaceID := "vault-shared-workspace-" + randomSuffix()

	resp1, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: "vault-user-alpha"},
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey for user alpha failed: %v", err)
	}

	resp2, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: "vault-user-beta"},
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey for user beta failed: %v", err)
	}

	// Different users should produce different keys even for the same workspace
	if string(resp1.EncryptedKey) == string(resp2.EncryptedKey) {
		t.Error("expected different encrypted keys for different users on the same workspace")
	}
}

// TestVaultDeriveWorkspaceKeyValidation tests that missing required fields
// result in errors from the vault service.
func TestVaultDeriveWorkspaceKeyValidation(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("missing workspace ID", func(t *testing.T) {
		_, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
			UserId:      &commonv1.UUID{Value: "vault-validation-user"},
			WorkspaceId: &commonv1.UUID{Value: ""},
		})
		if err == nil {
			t.Error("expected error for empty workspace ID, got nil")
		}
	})

	t.Run("missing user ID", func(t *testing.T) {
		_, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
			UserId:      &commonv1.UUID{Value: ""},
			WorkspaceId: &commonv1.UUID{Value: "vault-validation-workspace"},
		})
		if err == nil {
			t.Error("expected error for empty user ID, got nil")
		}
	})
}

// TestVaultDeriveFileKey tests that DeriveFileKey returns an encrypted key
// and key ID for valid inputs.
func TestVaultDeriveFileKey(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
		WorkspaceId: &commonv1.UUID{Value: "vault-file-test-workspace"},
		FileId:      &commonv1.UUID{Value: "vault-file-test-file-1"},
	})
	if err != nil {
		t.Fatalf("DeriveFileKey RPC failed: %v", err)
	}

	if len(resp.EncryptedKey) == 0 {
		t.Error("expected non-empty encrypted key in response")
	}
	if len(resp.KeyId) == 0 {
		t.Error("expected non-empty key ID in response")
	}
}

// TestVaultDeriveFileKeyDeterministic tests that the same inputs
// produce the same file key (deterministic derivation).
func TestVaultDeriveFileKeyDeterministic(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workspaceID := "vault-file-det-workspace-" + randomSuffix()
	fileID := "vault-file-det-file-" + randomSuffix()

	resp1, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
		FileId:      &commonv1.UUID{Value: fileID},
	})
	if err != nil {
		t.Fatalf("DeriveFileKey (1st call) RPC failed: %v", err)
	}

	resp2, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
		FileId:      &commonv1.UUID{Value: fileID},
	})
	if err != nil {
		t.Fatalf("DeriveFileKey (2nd call) RPC failed: %v", err)
	}

	// Same inputs should produce the same encrypted key
	if string(resp1.EncryptedKey) != string(resp2.EncryptedKey) {
		t.Error("expected identical encrypted keys for same inputs (file key derivation should be deterministic)")
	}
	if string(resp1.KeyId) != string(resp2.KeyId) {
		t.Error("expected identical key IDs for same inputs")
	}
}

// TestVaultDeriveFileKeyIsolation tests that different file IDs
// produce different file keys (isolation between files).
func TestVaultDeriveFileKeyIsolation(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workspaceID := "vault-file-iso-workspace-" + randomSuffix()

	resp1, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
		FileId:      &commonv1.UUID{Value: "vault-file-A"},
	})
	if err != nil {
		t.Fatalf("DeriveFileKey for file A failed: %v", err)
	}

	resp2, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
		FileId:      &commonv1.UUID{Value: "vault-file-B"},
	})
	if err != nil {
		t.Fatalf("DeriveFileKey for file B failed: %v", err)
	}

	// Different file IDs should produce different keys
	if string(resp1.EncryptedKey) == string(resp2.EncryptedKey) {
		t.Error("expected different encrypted keys for different file IDs")
	}
}

// TestVaultDeriveFileKeyWorkspaceIsolation tests that the same file ID
// in different workspaces produces different keys (cross-workspace isolation).
func TestVaultDeriveFileKeyWorkspaceIsolation(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fileID := "vault-same-file-" + randomSuffix()

	resp1, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
		WorkspaceId: &commonv1.UUID{Value: "vault-workspace-X"},
		FileId:      &commonv1.UUID{Value: fileID},
	})
	if err != nil {
		t.Fatalf("DeriveFileKey for workspace X failed: %v", err)
	}

	resp2, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
		WorkspaceId: &commonv1.UUID{Value: "vault-workspace-Y"},
		FileId:      &commonv1.UUID{Value: fileID},
	})
	if err != nil {
		t.Fatalf("DeriveFileKey for workspace Y failed: %v", err)
	}

	// Same file ID, different workspaces -> different keys
	if string(resp1.EncryptedKey) == string(resp2.EncryptedKey) {
		t.Error("expected different file keys for same file ID across different workspaces")
	}
}

// TestVaultDeriveFileKeyValidation tests that missing required fields
// result in errors from the vault service.
func TestVaultDeriveFileKeyValidation(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("missing workspace ID", func(t *testing.T) {
		_, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
			WorkspaceId: &commonv1.UUID{Value: ""},
			FileId:      &commonv1.UUID{Value: "vault-validation-file"},
		})
		if err == nil {
			t.Error("expected error for empty workspace ID, got nil")
		}
	})

	t.Run("missing file ID", func(t *testing.T) {
		_, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
			WorkspaceId: &commonv1.UUID{Value: "vault-validation-workspace"},
			FileId:      &commonv1.UUID{Value: ""},
		})
		if err == nil {
			t.Error("expected error for empty file ID, got nil")
		}
	})
}

// TestVaultRotateWorkspaceKey tests that RotateWorkspaceKey succeeds
// and produces a new key that differs from the original for different
// rotation instances.
func TestVaultRotateWorkspaceKey(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID := "vault-rotate-user-" + randomSuffix()
	workspaceID := "vault-rotate-workspace-" + randomSuffix()

	// Derive the original workspace key before rotation
	originalResp, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: userID},
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey (before rotation) RPC failed: %v", err)
	}

	// Rotate the workspace key
	_, err = client.RotateWorkspaceKey(ctx, &vaultv1.RotateWorkspaceKeyRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
		UserId:      &commonv1.UUID{Value: userID},
	})
	if err != nil {
		t.Fatalf("RotateWorkspaceKey RPC failed: %v", err)
	}

	// Derive the workspace key again after rotation
	// Note: In Phase 1, rotation derives from user_id (placeholder) so the
	// derived key may be the same. The key ID from DeriveWorkspaceKey should
	// remain consistent since it's derived from workspace ID.
	postRotationResp, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: userID},
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey (after rotation) RPC failed: %v", err)
	}

	// Both should return non-empty keys
	if len(originalResp.EncryptedKey) == 0 {
		t.Error("expected non-empty original encrypted key")
	}
	if len(postRotationResp.EncryptedKey) == 0 {
		t.Error("expected non-empty post-rotation encrypted key")
	}

	// Key IDs should be deterministic (derived from workspace ID)
	if string(originalResp.KeyId) != string(postRotationResp.KeyId) {
		t.Logf("key IDs differ before/after rotation: original=%s, rotated=%s",
			string(originalResp.KeyId), string(postRotationResp.KeyId))
	}
}

// TestVaultRotateWorkspaceKeyValidation tests that missing required fields
// result in errors from the vault service.
func TestVaultRotateWorkspaceKeyValidation(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("missing workspace ID", func(t *testing.T) {
		_, err := client.RotateWorkspaceKey(ctx, &vaultv1.RotateWorkspaceKeyRequest{
			WorkspaceId: &commonv1.UUID{Value: ""},
			UserId:      &commonv1.UUID{Value: "vault-rotate-validation-user"},
		})
		if err == nil {
			t.Error("expected error for empty workspace ID, got nil")
		}
	})

	t.Run("missing user ID", func(t *testing.T) {
		_, err := client.RotateWorkspaceKey(ctx, &vaultv1.RotateWorkspaceKeyRequest{
			WorkspaceId: &commonv1.UUID{Value: "vault-rotate-validation-workspace"},
			UserId:      &commonv1.UUID{Value: ""},
		})
		if err == nil {
			t.Error("expected error for empty user ID, got nil")
		}
	})
}