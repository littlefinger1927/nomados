package handler

import (
	"context"
	"testing"

	"github.com/nomados/nomados/packages/logging"
	"github.com/nomados/nomados/services/vault-service/internal/keyderivation"
)

// mockPublisher records published key rotation events for verification.
type mockPublisher struct {
	publishedEvents []keyRotatedEvent
	shouldError    bool
}

type keyRotatedEvent struct {
	workspaceID string
	keyID       string
}

func (m *mockPublisher) PublishKeyRotated(ctx context.Context, workspaceID string, keyID string) error {
	if m.shouldError {
		return nil // don't block on publisher errors
	}
	m.publishedEvents = append(m.publishedEvents, keyRotatedEvent{
		workspaceID: workspaceID,
		keyID:       keyID,
	})
	return nil
}

func newTestHandler(publisher KeyRotationPublisher) *VaultServiceHandler {
	kd := keyderivation.NewKeyDeriver()
	logger := logging.NewLogger("vault-service-test", nil)
	return NewVaultServiceHandler(kd, publisher, logger)
}

func TestDeriveWorkspaceKey(t *testing.T) {
	handler := newTestHandler(nil)

	t.Run("successful derivation", func(t *testing.T) {
		masterKey, err := handler.kd.DeriveMasterKey([]byte("test-password"), []byte("test-salt-16bytes"))
		if err != nil {
			t.Fatalf("failed to derive master key: %v", err)
		}

		req := &DeriveWorkspaceKeyRequest{
			MasterKey:   masterKey,
			WorkspaceID: "ws-123",
		}

		resp, err := handler.DeriveWorkspaceKey(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.WorkspaceKey) != keyderivation.KeyLength {
			t.Errorf("expected workspace key length %d, got %d", keyderivation.KeyLength, len(resp.WorkspaceKey))
		}
	})

	t.Run("zeroizes master key after operation", func(t *testing.T) {
		masterKey, err := handler.kd.DeriveMasterKey([]byte("test-password"), []byte("test-salt-16bytes"))
		if err != nil {
			t.Fatalf("failed to derive master key: %v", err)
		}

		req := &DeriveWorkspaceKeyRequest{
			MasterKey:   masterKey,
			WorkspaceID: "ws-456",
		}

		_, err = handler.DeriveWorkspaceKey(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// After the operation, master key should be zeroized.
		allZero := true
		for _, b := range req.MasterKey {
			if b != 0 {
				allZero = false
				break
			}
		}
		if !allZero {
			t.Errorf("master key should be zeroized after DeriveWorkspaceKey")
		}
	})

	t.Run("rejects empty master key", func(t *testing.T) {
		req := &DeriveWorkspaceKeyRequest{
			MasterKey:   []byte{},
			WorkspaceID: "ws-123",
		}

		_, err := handler.DeriveWorkspaceKey(context.Background(), req)
		if err == nil {
			t.Errorf("expected error for empty master key")
		}
	})

	t.Run("rejects empty workspace ID", func(t *testing.T) {
		req := &DeriveWorkspaceKeyRequest{
			MasterKey:   make([]byte, keyderivation.KeyLength),
			WorkspaceID: "",
		}

		_, err := handler.DeriveWorkspaceKey(context.Background(), req)
		if err == nil {
			t.Errorf("expected error for empty workspace ID")
		}
	})
}

func TestDeriveFileKey(t *testing.T) {
	handler := newTestHandler(nil)

	t.Run("successful derivation", func(t *testing.T) {
		masterKey, err := handler.kd.DeriveMasterKey([]byte("test-password"), []byte("test-salt-16bytes"))
		if err != nil {
			t.Fatalf("failed to derive master key: %v", err)
		}
		workspaceKey, err := handler.kd.DeriveWorkspaceKey(masterKey, []byte("ws-123"))
		if err != nil {
			t.Fatalf("failed to derive workspace key: %v", err)
		}

		req := &DeriveFileKeyRequest{
			WorkspaceKey: workspaceKey,
			FileID:       "file-456",
		}

		resp, err := handler.DeriveFileKey(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.FileKey) != keyderivation.KeyLength {
			t.Errorf("expected file key length %d, got %d", keyderivation.KeyLength, len(resp.FileKey))
		}
	})

	t.Run("zeroizes workspace key after operation", func(t *testing.T) {
		masterKey, err := handler.kd.DeriveMasterKey([]byte("test-password"), []byte("test-salt-16bytes"))
		if err != nil {
			t.Fatalf("failed to derive master key: %v", err)
		}
		workspaceKey, err := handler.kd.DeriveWorkspaceKey(masterKey, []byte("ws-789"))
		if err != nil {
			t.Fatalf("failed to derive workspace key: %v", err)
		}

		req := &DeriveFileKeyRequest{
			WorkspaceKey: workspaceKey,
			FileID:       "file-789",
		}

		_, err = handler.DeriveFileKey(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		allZero := true
		for _, b := range req.WorkspaceKey {
			if b != 0 {
				allZero = false
				break
			}
		}
		if !allZero {
			t.Errorf("workspace key should be zeroized after DeriveFileKey")
		}
	})

	t.Run("rejects empty workspace key", func(t *testing.T) {
		req := &DeriveFileKeyRequest{
			WorkspaceKey: []byte{},
			FileID:       "file-123",
		}

		_, err := handler.DeriveFileKey(context.Background(), req)
		if err == nil {
			t.Errorf("expected error for empty workspace key")
		}
	})

	t.Run("rejects empty file ID", func(t *testing.T) {
		req := &DeriveFileKeyRequest{
			WorkspaceKey: make([]byte, keyderivation.KeyLength),
			FileID:       "",
		}

		_, err := handler.DeriveFileKey(context.Background(), req)
		if err == nil {
			t.Errorf("expected error for empty file ID")
		}
	})
}

func TestRotateWorkspaceKey(t *testing.T) {
	publisher := &mockPublisher{}
	handler := newTestHandler(publisher)

	t.Run("successful rotation", func(t *testing.T) {
		masterKey, err := handler.kd.DeriveMasterKey([]byte("test-password"), []byte("test-salt-16bytes"))
		if err != nil {
			t.Fatalf("failed to derive master key: %v", err)
		}

		req := &RotateWorkspaceKeyRequest{
			MasterKey:   masterKey,
			WorkspaceID: "ws-rotate-1",
		}

		resp, err := handler.RotateWorkspaceKey(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.NewKey) != keyderivation.KeyLength {
			t.Errorf("expected key length %d, got %d", keyderivation.KeyLength, len(resp.NewKey))
		}
		if resp.KeyID == "" {
			t.Errorf("expected non-empty key ID")
		}
		if resp.RotatedAt.IsZero() {
			t.Errorf("expected non-zero rotated timestamp")
		}
	})

	t.Run("publishes rotation event", func(t *testing.T) {
		publisher.publishedEvents = nil
		masterKey, err := handler.kd.DeriveMasterKey([]byte("test-password"), []byte("test-salt-16bytes"))
		if err != nil {
			t.Fatalf("failed to derive master key: %v", err)
		}

		req := &RotateWorkspaceKeyRequest{
			MasterKey:   masterKey,
			WorkspaceID: "ws-rotate-2",
		}

		resp, err := handler.RotateWorkspaceKey(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(publisher.publishedEvents) != 1 {
			t.Fatalf("expected 1 published event, got %d", len(publisher.publishedEvents))
		}
		if publisher.publishedEvents[0].workspaceID != "ws-rotate-2" {
			t.Errorf("expected workspace ID ws-rotate-2, got %s", publisher.publishedEvents[0].workspaceID)
		}
		if publisher.publishedEvents[0].keyID != resp.KeyID {
			t.Errorf("expected key ID %s, got %s", resp.KeyID, publisher.publishedEvents[0].keyID)
		}
	})

	t.Run("zeroizes master key after operation", func(t *testing.T) {
		masterKey, err := handler.kd.DeriveMasterKey([]byte("test-password"), []byte("test-salt-16bytes"))
		if err != nil {
			t.Fatalf("failed to derive master key: %v", err)
		}

		req := &RotateWorkspaceKeyRequest{
			MasterKey:   masterKey,
			WorkspaceID: "ws-rotate-3",
		}

		_, err = handler.RotateWorkspaceKey(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		allZero := true
		for _, b := range req.MasterKey {
			if b != 0 {
				allZero = false
				break
			}
		}
		if !allZero {
			t.Errorf("master key should be zeroized after RotateWorkspaceKey")
		}
	})

	t.Run("rejects empty master key", func(t *testing.T) {
		req := &RotateWorkspaceKeyRequest{
			MasterKey:   []byte{},
			WorkspaceID: "ws-123",
		}

		_, err := handler.RotateWorkspaceKey(context.Background(), req)
		if err == nil {
			t.Errorf("expected error for empty master key")
		}
	})

	t.Run("rejects empty workspace ID", func(t *testing.T) {
		req := &RotateWorkspaceKeyRequest{
			MasterKey:   make([]byte, keyderivation.KeyLength),
			WorkspaceID: "",
		}

		_, err := handler.RotateWorkspaceKey(context.Background(), req)
		if err == nil {
			t.Errorf("expected error for empty workspace ID")
		}
	})
}