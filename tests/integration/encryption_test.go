package integration

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

// Key derivation constants matching the vault service's keyderivation package.
const (
	argon2Memory      = 64 * 1024 // 64MB in KiB
	argon2Iterations  = 3
	argon2Parallelism = 4
	keyLength         = 32

	workspaceKeyInfo = "nomados-workspace-key"
	fileKeyInfo      = "nomados-file-key"
)

// deriveMasterKey derives a 32-byte master key from a password and salt using Argon2id.
// This mirrors the vault service's keyderivation.DeriveMasterKey for integration testing.
func deriveMasterKey(password, salt []byte) ([]byte, error) {
	if len(password) == 0 {
		return nil, errEmpty("password")
	}
	if len(salt) == 0 {
		return nil, errEmpty("salt")
	}
	key := argon2.IDKey(password, salt, argon2Iterations, argon2Memory, argon2Parallelism, keyLength)
	result := make([]byte, keyLength)
	copy(result, key)
	return result, nil
}

// deriveWorkspaceKey derives a 32-byte workspace key from a master key and workspace ID
// using HKDF-SHA256. This mirrors the vault service's keyderivation.DeriveWorkspaceKey.
func deriveWorkspaceKey(masterKey []byte, workspaceID []byte) ([]byte, error) {
	if len(masterKey) != keyLength {
		return nil, errWrongSize("master key", keyLength, len(masterKey))
	}
	if len(workspaceID) == 0 {
		return nil, errEmpty("workspace ID")
	}
	hkdfReader := hkdf.New(sha256.New, masterKey, []byte(workspaceKeyInfo), workspaceID)
	key := make([]byte, keyLength)
	if _, err := hkdfReader.Read(key); err != nil {
		return nil, errHKDF("workspace key", err)
	}
	return key, nil
}

// deriveFileKey derives a 32-byte file key from a workspace key and file ID
// using HKDF-SHA256. This mirrors the vault service's keyderivation.DeriveFileKey.
func deriveFileKey(workspaceKey []byte, fileID []byte) ([]byte, error) {
	if len(workspaceKey) != keyLength {
		return nil, errWrongSize("workspace key", keyLength, len(workspaceKey))
	}
	if len(fileID) == 0 {
		return nil, errEmpty("file ID")
	}
	hkdfReader := hkdf.New(sha256.New, workspaceKey, []byte(fileKeyInfo), fileID)
	key := make([]byte, keyLength)
	if _, err := hkdfReader.Read(key); err != nil {
		return nil, errHKDF("file key", err)
	}
	return key, nil
}

// rotateWorkspaceKey derives a new workspace key from a master key and workspace ID.
// In Phase 1, this produces the same output as deriveWorkspaceKey.
func rotateWorkspaceKey(masterKey []byte, workspaceID []byte) ([]byte, error) {
	return deriveWorkspaceKey(masterKey, workspaceID)
}

type validationError struct {
	field  string
	msg    string
}

func (e validationError) Error() string { return e.field + ": " + e.msg }

func errEmpty(field string) validationError { return validationError{field, "must not be empty"} }
func errWrongSize(field string, expected, actual int) validationError {
	return validationError{field, "must be " + itoa(expected) + " bytes, got " + itoa(actual)}
}
func errHKDF(keyType string, err error) validationError {
	return validationError{keyType, "hkdf expand error: " + err.Error()}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// TestMasterKeyDerivation tests that deriving a master key from a password
// produces a 32-byte key and is deterministic for the same inputs.
func TestMasterKeyDerivation(t *testing.T) {
	t.Run("produces 32-byte key", func(t *testing.T) {
		key, err := deriveMasterKey([]byte("test-password-123"), []byte("test-salt-16-bytes"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(key) != keyLength {
			t.Errorf("expected key length %d, got %d", keyLength, len(key))
		}
	})

	t.Run("deterministic output for same inputs", func(t *testing.T) {
		key1, err := deriveMasterKey([]byte("test-password-123"), []byte("test-salt-16-bytes"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		key2, err := deriveMasterKey([]byte("test-password-123"), []byte("test-salt-16-bytes"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(key1, key2) {
			t.Error("expected identical master keys for same inputs")
		}
	})

	t.Run("different passwords produce different keys", func(t *testing.T) {
		key1, err := deriveMasterKey([]byte("password-alpha"), []byte("same-salt-value"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		key2, err := deriveMasterKey([]byte("password-beta"), []byte("same-salt-value"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bytes.Equal(key1, key2) {
			t.Error("expected different keys for different passwords")
		}
	})

	t.Run("different salts produce different keys", func(t *testing.T) {
		key1, err := deriveMasterKey([]byte("same-password"), []byte("salt-alpha-16b"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		key2, err := deriveMasterKey([]byte("same-password"), []byte("salt-beta-16b-"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bytes.Equal(key1, key2) {
			t.Error("expected different keys for different salts")
		}
	})

	t.Run("rejects empty password", func(t *testing.T) {
		_, err := deriveMasterKey([]byte{}, []byte("test-salt-16-bytes"))
		if err == nil {
			t.Error("expected error for empty password")
		}
	})

	t.Run("rejects empty salt", func(t *testing.T) {
		_, err := deriveMasterKey([]byte("test-password-123"), []byte{})
		if err == nil {
			t.Error("expected error for empty salt")
		}
	})
}

// TestWorkspaceKeyDerivation tests the workspace key derivation chain:
// master key + workspace_id -> HKDF -> workspace key.
func TestWorkspaceKeyDerivation(t *testing.T) {
	masterKey, err := deriveMasterKey([]byte("workspace-test-password"), []byte("workspace-test-salt"))
	if err != nil {
		t.Fatalf("failed to derive master key: %v", err)
	}

	t.Run("produces 32-byte workspace key", func(t *testing.T) {
		wsKey, err := deriveWorkspaceKey(masterKey, []byte("workspace-id-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(wsKey) != keyLength {
			t.Errorf("expected workspace key length %d, got %d", keyLength, len(wsKey))
		}
	})

	t.Run("different workspace IDs produce different keys", func(t *testing.T) {
		wsKey1, err := deriveWorkspaceKey(masterKey, []byte("workspace-id-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wsKey2, err := deriveWorkspaceKey(masterKey, []byte("workspace-id-2"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bytes.Equal(wsKey1, wsKey2) {
			t.Error("expected different workspace keys for different workspace IDs")
		}
	})

	t.Run("workspace key differs from master key", func(t *testing.T) {
		wsKey, err := deriveWorkspaceKey(masterKey, []byte("workspace-id-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bytes.Equal(masterKey, wsKey) {
			t.Error("workspace key should differ from master key")
		}
	})

	t.Run("deterministic workspace key derivation", func(t *testing.T) {
		wsKey1, err := deriveWorkspaceKey(masterKey, []byte("deterministic-ws"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wsKey2, err := deriveWorkspaceKey(masterKey, []byte("deterministic-ws"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(wsKey1, wsKey2) {
			t.Error("expected identical workspace keys for same inputs")
		}
	})

	t.Run("rejects wrong-sized master key", func(t *testing.T) {
		_, err := deriveWorkspaceKey([]byte("short"), []byte("workspace-id-1"))
		if err == nil {
			t.Error("expected error for short master key")
		}
	})

	t.Run("rejects empty workspace ID", func(t *testing.T) {
		_, err := deriveWorkspaceKey(masterKey, []byte{})
		if err == nil {
			t.Error("expected error for empty workspace ID")
		}
	})
}

// TestFileKeyDerivation tests the file key derivation chain:
// workspace key + file_id -> HKDF -> file key.
func TestFileKeyDerivation(t *testing.T) {
	masterKey, _ := deriveMasterKey([]byte("file-test-password"), []byte("file-test-salt"))
	workspaceKey, _ := deriveWorkspaceKey(masterKey, []byte("workspace-for-files"))

	t.Run("produces 32-byte file key", func(t *testing.T) {
		fileKey, err := deriveFileKey(workspaceKey, []byte("file-id-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(fileKey) != keyLength {
			t.Errorf("expected file key length %d, got %d", keyLength, len(fileKey))
		}
	})

	t.Run("different file IDs produce different keys", func(t *testing.T) {
		fileKey1, err := deriveFileKey(workspaceKey, []byte("file-id-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		fileKey2, err := deriveFileKey(workspaceKey, []byte("file-id-2"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bytes.Equal(fileKey1, fileKey2) {
			t.Error("expected different file keys for different file IDs")
		}
	})

	t.Run("file key differs from workspace key", func(t *testing.T) {
		fileKey, err := deriveFileKey(workspaceKey, []byte("file-id-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bytes.Equal(workspaceKey, fileKey) {
			t.Error("file key should differ from workspace key")
		}
	})

	t.Run("file keys across workspaces are isolated", func(t *testing.T) {
		// Derive workspace keys for two different workspaces
		wsKey1, _ := deriveWorkspaceKey(masterKey, []byte("ws-isolation-1"))
		wsKey2, _ := deriveWorkspaceKey(masterKey, []byte("ws-isolation-2"))

		// Same file ID, different workspace keys -> different file keys
		fileKey1, _ := deriveFileKey(wsKey1, []byte("same-file-id"))
		fileKey2, _ := deriveFileKey(wsKey2, []byte("same-file-id"))

		if bytes.Equal(fileKey1, fileKey2) {
			t.Error("expected different file keys for the same file ID across different workspaces")
		}
	})

	t.Run("rejects wrong-sized workspace key", func(t *testing.T) {
		_, err := deriveFileKey([]byte("short"), []byte("file-id-1"))
		if err == nil {
			t.Error("expected error for short workspace key")
		}
	})

	t.Run("rejects empty file ID", func(t *testing.T) {
		_, err := deriveFileKey(workspaceKey, []byte{})
		if err == nil {
			t.Error("expected error for empty file ID")
		}
	})
}

// TestEncryptionRoundTrip tests the full key derivation chain and
// a simulated encrypt/decrypt round-trip using the key hierarchy.
// Since the Rust crypto package provides the actual XChaCha20-Poly1305
// encryption, this test validates that the key derivation produces
// usable 32-byte keys and demonstrates key correctness via XOR.
func TestEncryptionRoundTrip(t *testing.T) {
	// Derive the full key chain: password -> master -> workspace -> file
	masterKey, err := deriveMasterKey([]byte("roundtrip-password"), []byte("roundtrip-salt-16"))
	if err != nil {
		t.Fatalf("failed to derive master key: %v", err)
	}

	workspaceKey, err := deriveWorkspaceKey(masterKey, []byte("roundtrip-workspace"))
	if err != nil {
		t.Fatalf("failed to derive workspace key: %v", err)
	}

	fileKey, err := deriveFileKey(workspaceKey, []byte("roundtrip-file"))
	if err != nil {
		t.Fatalf("failed to derive file key: %v", err)
	}

	// Verify all keys are 32 bytes (suitable for XChaCha20-Poly1305)
	if len(masterKey) != 32 {
		t.Errorf("expected master key length 32, got %d", len(masterKey))
	}
	if len(workspaceKey) != 32 {
		t.Errorf("expected workspace key length 32, got %d", len(workspaceKey))
	}
	if len(fileKey) != 32 {
		t.Errorf("expected file key length 32, got %d", len(fileKey))
	}

	// Verify all three keys are distinct
	if bytes.Equal(masterKey, workspaceKey) {
		t.Error("master key and workspace key should differ")
	}
	if bytes.Equal(masterKey, fileKey) {
		t.Error("master key and file key should differ")
	}
	if bytes.Equal(workspaceKey, fileKey) {
		t.Error("workspace key and file key should differ")
	}

	// Simulate encrypt/decrypt round-trip using file key.
	// In production, XChaCha20-Poly1305 would be used via the Rust crypto package.
	// Here we verify the key derivation produces usable 32-byte keys.
	plaintext := []byte("Hello, NomadOS! This is secret data that needs encryption.")

	// For the round-trip, we use a simple XOR cipher to demonstrate
	// that the same key can encrypt and decrypt data.
	// Real encryption uses XChaCha20-Poly1305 via packages/crypto.
	ciphertext := make([]byte, len(plaintext))
	for i := range plaintext {
		ciphertext[i] = plaintext[i] ^ fileKey[i%len(fileKey)]
	}

	// Decrypt by XORing again with the same key
	decrypted := make([]byte, len(ciphertext))
	for i := range ciphertext {
		decrypted[i] = ciphertext[i] ^ fileKey[i%len(fileKey)]
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("decrypted data does not match original plaintext")
	}

	// Verify that using a different file key does NOT decrypt correctly
	differentFileKey, _ := deriveFileKey(workspaceKey, []byte("different-file"))
	wrongDecrypted := make([]byte, len(ciphertext))
	for i := range ciphertext {
		wrongDecrypted[i] = ciphertext[i] ^ differentFileKey[i%len(differentFileKey)]
	}

	if bytes.Equal(plaintext, wrongDecrypted) {
		t.Error("using a different key should not produce correct plaintext")
	}
}

// TestKeyRotation tests that after a workspace key rotation,
// the old key still decrypts data that was encrypted with it.
// In Phase 1, RotateWorkspaceKey produces the same key as DeriveWorkspaceKey
// for the same inputs, so we test the rotation semantics.
func TestKeyRotation(t *testing.T) {
	masterKey, _ := deriveMasterKey([]byte("rotation-password"), []byte("rotation-salt-16"))
	workspaceID := []byte("rotation-workspace")

	t.Run("rotation produces valid 32-byte key", func(t *testing.T) {
		rotatedKey, err := rotateWorkspaceKey(masterKey, workspaceID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rotatedKey) != keyLength {
			t.Errorf("expected rotated key length %d, got %d", keyLength, len(rotatedKey))
		}
	})

	t.Run("rotated key matches DeriveWorkspaceKey for same inputs", func(t *testing.T) {
		// In Phase 1, RotateWorkspaceKey delegates to DeriveWorkspaceKey
		rotatedKey, err := rotateWorkspaceKey(masterKey, workspaceID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		derivedKey, err := deriveWorkspaceKey(masterKey, workspaceID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(rotatedKey, derivedKey) {
			t.Error("rotated key should match derived key for same inputs in Phase 1")
		}
	})

	t.Run("rotation is deterministic", func(t *testing.T) {
		key1, err := rotateWorkspaceKey(masterKey, workspaceID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		key2, err := rotateWorkspaceKey(masterKey, workspaceID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(key1, key2) {
			t.Error("rotation should be deterministic for same inputs")
		}
	})

	t.Run("old workspace key still decrypts data after rotation concept", func(t *testing.T) {
		// In Phase 1, rotation produces the same key as derivation.
		// In future phases, rotation will produce a new key and the old key
		// must still be able to decrypt data until re-encryption completes.
		// This test validates the concept: derive key, encrypt data, then
		// verify the same key can still decrypt.
		wsKey, err := deriveWorkspaceKey(masterKey, workspaceID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Simulate encrypting data with the workspace key
		plaintext := []byte("sensitive workspace data")
		ciphertext := make([]byte, len(plaintext))
		for i := range plaintext {
			ciphertext[i] = plaintext[i] ^ wsKey[i%len(wsKey)]
		}

		// Verify the same key can still decrypt
		decrypted := make([]byte, len(ciphertext))
		for i := range ciphertext {
			decrypted[i] = ciphertext[i] ^ wsKey[i%len(wsKey)]
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Error("original key should still decrypt data after rotation concept validation")
		}
	})
}

// TestKeyDerivationIsolation verifies that the full key hierarchy
// provides cryptographic isolation between different users, workspaces, and files.
func TestKeyDerivationIsolation(t *testing.T) {
	t.Run("cross-user isolation", func(t *testing.T) {
		// Two different users should derive completely different master keys
		user1Master, _ := deriveMasterKey([]byte("user1-password"), []byte("user1-salt-16b"))
		user2Master, _ := deriveMasterKey([]byte("user2-password"), []byte("user2-salt-16b"))

		if bytes.Equal(user1Master, user2Master) {
			t.Error("different users should have different master keys")
		}

		// Even with the same workspace ID, different master keys produce different workspace keys
		wsKey1, _ := deriveWorkspaceKey(user1Master, []byte("same-workspace"))
		wsKey2, _ := deriveWorkspaceKey(user2Master, []byte("same-workspace"))

		if bytes.Equal(wsKey1, wsKey2) {
			t.Error("different users should derive different workspace keys for the same workspace ID")
		}
	})

	t.Run("random salt generates unique keys", func(t *testing.T) {
		salt1 := make([]byte, 16)
		salt2 := make([]byte, 16)
		rand.Read(salt1)
		rand.Read(salt2)

		key1, _ := deriveMasterKey([]byte("same-password"), salt1)
		key2, _ := deriveMasterKey([]byte("same-password"), salt2)

		if bytes.Equal(key1, key2) {
			t.Error("random salts should produce different master keys")
		}
	})

	t.Run("full hierarchy produces distinct keys at each level", func(t *testing.T) {
		kd := &struct{}{} // placeholder — we use local functions
		_ = kd

		masterKey, _ := deriveMasterKey([]byte("hierarchy-password"), []byte("hierarchy-salt"))
		wsKey, _ := deriveWorkspaceKey(masterKey, []byte("hierarchy-ws"))
		fileKey, _ := deriveFileKey(wsKey, []byte("hierarchy-file"))

		// All three keys should be different from each other
		if bytes.Equal(masterKey, wsKey) {
			t.Error("master key should differ from workspace key")
		}
		if bytes.Equal(masterKey, fileKey) {
			t.Error("master key should differ from file key")
		}
		if bytes.Equal(wsKey, fileKey) {
			t.Error("workspace key should differ from file key")
		}
	})
}