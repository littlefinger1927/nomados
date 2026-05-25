package keyderivation

import (
	"bytes"
	"testing"
)

func TestDeriveMasterKey(t *testing.T) {
	kd := NewKeyDeriver()

	t.Run("produces 32-byte key", func(t *testing.T) {
		key, err := kd.DeriveMasterKey([]byte("password123"), []byte("somesalt12345678"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(key) != KeyLength {
			t.Errorf("expected key length %d, got %d", KeyLength, len(key))
		}
	})

	t.Run("deterministic output for same inputs", func(t *testing.T) {
		key1, err := kd.DeriveMasterKey([]byte("password123"), []byte("somesalt12345678"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		key2, err := kd.DeriveMasterKey([]byte("password123"), []byte("somesalt12345678"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(key1, key2) {
			t.Errorf("expected identical keys for same inputs")
		}
	})

	t.Run("different passwords produce different keys", func(t *testing.T) {
		key1, err := kd.DeriveMasterKey([]byte("password1"), []byte("somesalt12345678"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		key2, err := kd.DeriveMasterKey([]byte("password2"), []byte("somesalt12345678"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bytes.Equal(key1, key2) {
			t.Errorf("expected different keys for different passwords")
		}
	})

	t.Run("different salts produce different keys", func(t *testing.T) {
		key1, err := kd.DeriveMasterKey([]byte("password123"), []byte("saltA1234567890"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		key2, err := kd.DeriveMasterKey([]byte("password123"), []byte("saltB1234567890"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bytes.Equal(key1, key2) {
			t.Errorf("expected different keys for different salts")
		}
	})

	t.Run("rejects empty password", func(t *testing.T) {
		_, err := kd.DeriveMasterKey([]byte{}, []byte("somesalt12345678"))
		if err == nil {
			t.Errorf("expected error for empty password")
		}
	})

	t.Run("rejects empty salt", func(t *testing.T) {
		_, err := kd.DeriveMasterKey([]byte("password123"), []byte{})
		if err == nil {
			t.Errorf("expected error for empty salt")
		}
	})
}

func TestDeriveWorkspaceKey(t *testing.T) {
	kd := NewKeyDeriver()
	masterKey, _ := kd.DeriveMasterKey([]byte("password123"), []byte("somesalt12345678"))

	t.Run("produces 32-byte key", func(t *testing.T) {
		key, err := kd.DeriveWorkspaceKey(masterKey, []byte("workspace-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(key) != KeyLength {
			t.Errorf("expected key length %d, got %d", KeyLength, len(key))
		}
	})

	t.Run("deterministic output for same inputs", func(t *testing.T) {
		key1, err := kd.DeriveWorkspaceKey(masterKey, []byte("workspace-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		key2, err := kd.DeriveWorkspaceKey(masterKey, []byte("workspace-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(key1, key2) {
			t.Errorf("expected identical keys for same inputs")
		}
	})

	t.Run("different workspace IDs produce different keys", func(t *testing.T) {
		key1, err := kd.DeriveWorkspaceKey(masterKey, []byte("workspace-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		key2, err := kd.DeriveWorkspaceKey(masterKey, []byte("workspace-2"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bytes.Equal(key1, key2) {
			t.Errorf("expected different keys for different workspace IDs")
		}
	})

	t.Run("rejects wrong-sized master key", func(t *testing.T) {
		_, err := kd.DeriveWorkspaceKey([]byte("short"), []byte("workspace-1"))
		if err == nil {
			t.Errorf("expected error for short master key")
		}
	})

	t.Run("rejects empty workspace ID", func(t *testing.T) {
		_, err := kd.DeriveWorkspaceKey(masterKey, []byte{})
		if err == nil {
			t.Errorf("expected error for empty workspace ID")
		}
	})
}

func TestDeriveFileKey(t *testing.T) {
	kd := NewKeyDeriver()
	masterKey, _ := kd.DeriveMasterKey([]byte("password123"), []byte("somesalt12345678"))
	workspaceKey, _ := kd.DeriveWorkspaceKey(masterKey, []byte("workspace-1"))

	t.Run("produces 32-byte key", func(t *testing.T) {
		key, err := kd.DeriveFileKey(workspaceKey, []byte("file-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(key) != KeyLength {
			t.Errorf("expected key length %d, got %d", KeyLength, len(key))
		}
	})

	t.Run("deterministic output for same inputs", func(t *testing.T) {
		key1, err := kd.DeriveFileKey(workspaceKey, []byte("file-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		key2, err := kd.DeriveFileKey(workspaceKey, []byte("file-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(key1, key2) {
			t.Errorf("expected identical keys for same inputs")
		}
	})

	t.Run("different file IDs produce different keys", func(t *testing.T) {
		key1, err := kd.DeriveFileKey(workspaceKey, []byte("file-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		key2, err := kd.DeriveFileKey(workspaceKey, []byte("file-2"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bytes.Equal(key1, key2) {
			t.Errorf("expected different keys for different file IDs")
		}
	})

	t.Run("rejects wrong-sized workspace key", func(t *testing.T) {
		_, err := kd.DeriveFileKey([]byte("short"), []byte("file-1"))
		if err == nil {
			t.Errorf("expected error for short workspace key")
		}
	})

	t.Run("rejects empty file ID", func(t *testing.T) {
		_, err := kd.DeriveFileKey(workspaceKey, []byte{})
		if err == nil {
			t.Errorf("expected error for empty file ID")
		}
	})
}

func TestKeyHierarchyIsolation(t *testing.T) {
	kd := NewKeyDeriver()

	// Verify the full key hierarchy: password -> master -> workspace -> file
	// produces keys that are all distinct.
	masterKey, err := kd.DeriveMasterKey([]byte("password123"), []byte("somesalt12345678"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	workspaceKey, err := kd.DeriveWorkspaceKey(masterKey, []byte("workspace-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fileKey, err := kd.DeriveFileKey(workspaceKey, []byte("file-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All three keys should be different from each other
	if bytes.Equal(masterKey, workspaceKey) {
		t.Errorf("master key should differ from workspace key")
	}
	if bytes.Equal(masterKey, fileKey) {
		t.Errorf("master key should differ from file key")
	}
	if bytes.Equal(workspaceKey, fileKey) {
		t.Errorf("workspace key should differ from file key")
	}
}

func TestRotateWorkspaceKey(t *testing.T) {
	kd := NewKeyDeriver()
	masterKey, _ := kd.DeriveMasterKey([]byte("password123"), []byte("somesalt12345678"))

	t.Run("produces same output as DeriveWorkspaceKey", func(t *testing.T) {
		wsKey, err := kd.DeriveWorkspaceKey(masterKey, []byte("workspace-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		rotatedKey, err := kd.RotateWorkspaceKey(masterKey, []byte("workspace-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(wsKey, rotatedKey) {
			t.Errorf("RotateWorkspaceKey should produce same key as DeriveWorkspaceKey for same inputs")
		}
	})

	t.Run("produces 32-byte key", func(t *testing.T) {
		key, err := kd.RotateWorkspaceKey(masterKey, []byte("workspace-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(key) != KeyLength {
			t.Errorf("expected key length %d, got %d", KeyLength, len(key))
		}
	})
}

func TestZeroize(t *testing.T) {
	t.Run("clears buffer contents", func(t *testing.T) {
		buffer := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
			17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
		Zeroize(buffer)
		expected := make([]byte, 32)
		if !bytes.Equal(buffer, expected) {
			t.Errorf("expected buffer to be zeroed, got %v", buffer)
		}
	})

	t.Run("handles empty buffer", func(t *testing.T) {
		buffer := []byte{}
		Zeroize(buffer) // should not panic
	})

	t.Run("handles nil buffer", func(t *testing.T) {
		var buffer []byte
		Zeroize(buffer) // should not panic
	})
}

func TestCustomArgon2Params(t *testing.T) {
	kd := NewKeyDeriverWithParams(32*1024, 1, 2)

	key, err := kd.DeriveMasterKey([]byte("password123"), []byte("somesalt12345678"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != KeyLength {
		t.Errorf("expected key length %d, got %d", KeyLength, len(key))
	}

	// Different params should produce different keys from same inputs
	defaultKD := NewKeyDeriver()
	defaultKey, _ := defaultKD.DeriveMasterKey([]byte("password123"), []byte("somesalt12345678"))
	if bytes.Equal(key, defaultKey) {
		t.Errorf("different Argon2id params should produce different keys")
	}
}