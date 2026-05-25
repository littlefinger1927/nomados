package keyderivation

import (
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

// TODO: migrate to Rust crypto via CGo for interop with packages/crypto.
// The Go implementation uses the same algorithms (Argon2id, HKDF-SHA256)
// and must produce identical output for the same inputs.

// Argon2id parameters matching the Rust crypto package (packages/crypto/src/kdf.rs).
const (
	Argon2Memory      = 64 * 1024 // 64MB in KiB
	Argon2Iterations  = 3
	Argon2Parallelism = 4
	KeyLength         = 32
)

// HKDF info strings matching the Rust crypto package.
var (
	workspaceKeyInfo = []byte("nomados-workspace-key")
	fileKeyInfo       = []byte("nomados-file-key")
)

// KeyDeriver provides deterministic key derivation functions following
// the NomadOS encryption key hierarchy:
//
//	User Password + Salt → Argon2id → Master Key (32 bytes)
//	Master Key + workspace_id → HKDF-SHA256 → Workspace Key (32 bytes)
//	Workspace Key + file_id → HKDF-SHA256 → File Key (32 bytes)
//	File Key → XChaCha20-Poly1305 → Encrypted Data
type KeyDeriver struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

// NewKeyDeriver creates a KeyDeriver with default Argon2id parameters.
func NewKeyDeriver() *KeyDeriver {
	return &KeyDeriver{
		memory:      Argon2Memory,
		iterations:  Argon2Iterations,
		parallelism: Argon2Parallelism,
	}
}

// NewKeyDeriverWithParams creates a KeyDeriver with custom Argon2id parameters.
func NewKeyDeriverWithParams(memory uint32, iterations uint32, parallelism uint8) *KeyDeriver {
	return &KeyDeriver{
		memory:      memory,
		iterations:  iterations,
		parallelism: parallelism,
	}
}

// DeriveMasterKey derives a 32-byte master key from a password and salt using Argon2id.
// The parameters match the Rust crypto package for cross-language interop.
func (kd *KeyDeriver) DeriveMasterKey(password, salt []byte) ([]byte, error) {
	if len(password) == 0 {
		return nil, fmt.Errorf("password must not be empty")
	}
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt must not be empty")
	}

	key := argon2.IDKey(password, salt, kd.iterations, kd.memory, kd.parallelism, KeyLength)
	result := make([]byte, KeyLength)
	copy(result, key)
	return result, nil
}

// DeriveWorkspaceKey derives a 32-byte workspace key from a master key and workspace ID
// using HKDF-SHA256 with info="nomados-workspace-key" and workspace_id as the salt-like
// input (passed as HKDF info context per the Rust implementation).
//
// In the Rust implementation, Hkdf::new(Some(b"nomados-workspace-key"), master_key) sets
// the info as the HKDF salt and then hkdf.expand(workspace_id, &mut key) uses workspace_id
// as the expand info. We replicate this exactly.
func (kd *KeyDeriver) DeriveWorkspaceKey(masterKey []byte, workspaceID []byte) ([]byte, error) {
	if len(masterKey) != KeyLength {
		return nil, fmt.Errorf("master key must be %d bytes, got %d", KeyLength, len(masterKey))
	}
	if len(workspaceID) == 0 {
		return nil, fmt.Errorf("workspace ID must not be empty")
	}

	hkdfReader := hkdf.New(sha256.New, masterKey, workspaceKeyInfo, workspaceID)
	key := make([]byte, KeyLength)
	if _, err := hkdfReader.Read(key); err != nil {
		return nil, fmt.Errorf("hkdf expand error: %w", err)
	}
	return key, nil
}

// DeriveFileKey derives a 32-byte file key from a workspace key and file ID
// using HKDF-SHA256 with info="nomados-file-key" and file_id as the expand info,
// matching the Rust crypto package's implementation.
func (kd *KeyDeriver) DeriveFileKey(workspaceKey []byte, fileID []byte) ([]byte, error) {
	if len(workspaceKey) != KeyLength {
		return nil, fmt.Errorf("workspace key must be %d bytes, got %d", KeyLength, len(workspaceKey))
	}
	if len(fileID) == 0 {
		return nil, fmt.Errorf("file ID must not be empty")
	}

	hkdfReader := hkdf.New(sha256.New, workspaceKey, fileKeyInfo, fileID)
	key := make([]byte, KeyLength)
	if _, err := hkdfReader.Read(key); err != nil {
		return nil, fmt.Errorf("hkdf expand error: %w", err)
	}
	return key, nil
}

// RotateWorkspaceKey derives a new workspace key from a master key and workspace ID.
// This produces the same output as DeriveWorkspaceKey but is semantically distinct:
// the caller should track the old key for re-encryption of file keys until rotation
// completes. The rotation event is published by the handler, not here.
func (kd *KeyDeriver) RotateWorkspaceKey(currentMasterKey []byte, workspaceID []byte) ([]byte, error) {
	return kd.DeriveWorkspaceKey(currentMasterKey, workspaceID)
}

// Zeroize securely clears a byte buffer by overwriting it with zeros.
// This should be called after key material is no longer needed.
func Zeroize(buffer []byte) {
	for i := range buffer {
		buffer[i] = 0
	}
}