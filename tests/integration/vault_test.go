package integration

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
	vaultv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/vault/v1"
)

// TestVaultDeriveWorkspaceKey tests that DeriveWorkspaceKey returns a non-empty
// encrypted key and key ID for valid inputs.
func TestVaultDeriveWorkspaceKey(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID := "vault-test-user-" + randomSuffix()
	workspaceID := "vault-test-ws-" + randomSuffix()

	resp, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: userID},
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey RPC failed: %v", err)
	}

	if len(resp.EncryptedKey) == 0 {
		t.Error("expected non-empty encrypted_key in DeriveWorkspaceKey response")
	}
	if len(resp.KeyId) == 0 {
		t.Error("expected non-empty key_id in DeriveWorkspaceKey response")
	}
}

// TestVaultDeriveWorkspaceKeyDeterministic tests that calling DeriveWorkspaceKey
// with the same inputs produces the same encrypted key.
func TestVaultDeriveWorkspaceKeyDeterministic(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID := "vault-det-user-" + randomSuffix()
	workspaceID := "vault-det-ws-" + randomSuffix()

	resp1, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: userID},
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey (call 1) RPC failed: %v", err)
	}

	resp2, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: userID},
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey (call 2) RPC failed: %v", err)
	}

	if len(resp1.EncryptedKey) != len(resp2.EncryptedKey) {
		t.Errorf("expected same key length for deterministic inputs, got %d vs %d",
			len(resp1.EncryptedKey), len(resp2.EncryptedKey))
	}

	// Compare byte-by-byte
	for i := range resp1.EncryptedKey {
		if resp1.EncryptedKey[i] != resp2.EncryptedKey[i] {
			t.Error("expected identical encrypted keys for same inputs")
			break
		}
	}
}

// TestVaultDeriveWorkspaceKeyIsolation tests that different workspace IDs
// produce different encrypted keys, even for the same user.
func TestVaultDeriveWorkspaceKeyIsolation(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID := "vault-iso-user-" + randomSuffix()

	resp1, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: userID},
		WorkspaceId: &commonv1.UUID{Value: "workspace-alpha"},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey (workspace-alpha) RPC failed: %v", err)
	}

	resp2, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
		UserId:      &commonv1.UUID{Value: userID},
		WorkspaceId: &commonv1.UUID{Value: "workspace-beta"},
	})
	if err != nil {
		t.Fatalf("DeriveWorkspaceKey (workspace-beta) RPC failed: %v", err)
	}

	// Different workspace IDs should produce different keys for the same user
	if len(resp1.EncryptedKey) == 0 || len(resp2.EncryptedKey) == 0 {
		t.Fatal("expected non-empty encrypted keys for both workspace isolation requests")
	}

	keysMatch := true
	if len(resp1.EncryptedKey) != len(resp2.EncryptedKey) {
		keysMatch = false
	} else {
		for i := range resp1.EncryptedKey {
			if resp1.EncryptedKey[i] != resp2.EncryptedKey[i] {
				keysMatch = false
				break
			}
		}
	}
	if keysMatch {
		t.Error("expected different encrypted keys for different workspace IDs")
	}
}

// TestVaultDeriveFileKey tests that DeriveFileKey returns a non-empty
// encrypted key and key ID for valid inputs.
func TestVaultDeriveFileKey(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workspaceID := "vault-file-ws-" + randomSuffix()
	fileID := "vault-file-id-" + randomSuffix()

	resp, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
		FileId:      &commonv1.UUID{Value: fileID},
	})
	if err != nil {
		t.Fatalf("DeriveFileKey RPC failed: %v", err)
	}

	if len(resp.EncryptedKey) == 0 {
		t.Error("expected non-empty encrypted_key in DeriveFileKey response")
	}
	if len(resp.KeyId) == 0 {
		t.Error("expected non-empty key_id in DeriveFileKey response")
	}
}

// TestVaultDeriveFileKeyDeterministic tests that calling DeriveFileKey
// with the same inputs produces the same encrypted file key.
func TestVaultDeriveFileKeyDeterministic(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workspaceID := "vault-file-det-ws-" + randomSuffix()
	fileID := "vault-file-det-id-" + randomSuffix()

	resp1, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
		FileId:      &commonv1.UUID{Value: fileID},
	})
	if err != nil {
		t.Fatalf("DeriveFileKey (call 1) RPC failed: %v", err)
	}

	resp2, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
		FileId:      &commonv1.UUID{Value: fileID},
	})
	if err != nil {
		t.Fatalf("DeriveFileKey (call 2) RPC failed: %v", err)
	}

	if len(resp1.EncryptedKey) != len(resp2.EncryptedKey) {
		t.Errorf("expected same key length for deterministic inputs, got %d vs %d",
			len(resp1.EncryptedKey), len(resp2.EncryptedKey))
	}

	for i := range resp1.EncryptedKey {
		if resp1.EncryptedKey[i] != resp2.EncryptedKey[i] {
			t.Error("expected identical file keys for same inputs")
			break
		}
	}
}

// TestVaultRotateWorkspaceKey tests that key rotation succeeds and returns
// a valid empty response (the proto returns common.v1.Empty on success).
func TestVaultRotateWorkspaceKey(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workspaceID := "vault-rotate-ws-" + randomSuffix()
	userID := "vault-rotate-user-" + randomSuffix()

	_, err := client.RotateWorkspaceKey(ctx, &vaultv1.RotateWorkspaceKeyRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
		UserId:      &commonv1.UUID{Value: userID},
	})
	if err != nil {
		t.Fatalf("RotateWorkspaceKey RPC failed: %v", err)
	}
}

// TestVaultDeriveWorkspaceKeyValidation tests that missing required fields
// are rejected by the vault service.
func TestVaultDeriveWorkspaceKeyValidation(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("missing workspace ID returns error", func(t *testing.T) {
		_, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
			UserId:      &commonv1.UUID{Value: "validation-user"},
			WorkspaceId: &commonv1.UUID{Value: ""}, // empty workspace ID
		})
		if err == nil {
			t.Error("expected error for empty workspace ID, got nil")
		}
	})

	t.Run("missing user ID returns error", func(t *testing.T) {
		_, err := client.DeriveWorkspaceKey(ctx, &vaultv1.DeriveWorkspaceKeyRequest{
			UserId:      &commonv1.UUID{Value: ""}, // empty user ID
			WorkspaceId: &commonv1.UUID{Value: "validation-workspace"},
		})
		if err == nil {
			t.Error("expected error for empty user ID, got nil")
		}
	})
}

// TestVaultDeriveFileKeyValidation tests that missing required fields
// are rejected by the vault service.
func TestVaultDeriveFileKeyValidation(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("missing workspace ID returns error", func(t *testing.T) {
		_, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
			WorkspaceId: &commonv1.UUID{Value: ""},
			FileId:      &commonv1.UUID{Value: "validation-file"},
		})
		if err == nil {
			t.Error("expected error for empty workspace ID, got nil")
		}
	})

	t.Run("missing file ID returns error", func(t *testing.T) {
		_, err := client.DeriveFileKey(ctx, &vaultv1.DeriveFileKeyRequest{
			WorkspaceId: &commonv1.UUID{Value: "validation-workspace"},
			FileId:      &commonv1.UUID{Value: ""},
		})
		if err == nil {
			t.Error("expected error for empty file ID, got nil")
		}
	})
}

// TestVaultRotateWorkspaceKeyValidation tests that missing required fields
// are rejected by the vault service.
func TestVaultRotateWorkspaceKeyValidation(t *testing.T) {
	skipIfUnreachable(t, vaultServiceAddr, "vault")

	client, conn := newVaultClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("missing workspace ID returns error", func(t *testing.T) {
		_, err := client.RotateWorkspaceKey(ctx, &vaultv1.RotateWorkspaceKeyRequest{
			WorkspaceId: &commonv1.UUID{Value: ""},
			UserId:      &commonv1.UUID{Value: "validation-user"},
		})
		if err == nil {
			t.Error("expected error for empty workspace ID, got nil")
		}
	})

	t.Run("missing user ID returns error", func(t *testing.T) {
		_, err := client.RotateWorkspaceKey(ctx, &vaultv1.RotateWorkspaceKeyRequest{
			WorkspaceId: &commonv1.UUID{Value: "validation-workspace"},
			UserId:      &commonv1.UUID{Value: ""},
		})
		if err == nil {
			t.Error("expected error for empty user ID, got nil")
		}
	})
}